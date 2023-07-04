package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/bluesky-social/indigo/carstore"
	"github.com/bluesky-social/indigo/pds"
	"github.com/bluesky-social/indigo/plc"
	"github.com/bluesky-social/indigo/supercollider"
	"github.com/bluesky-social/indigo/testing"
	"github.com/whyrusleeping/go-did"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Trap SIGINT to trigger a shutdown.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-signals:
			cancel()
			fmt.Println("shutting down on signal")
			// Give the server some time to shutdown gracefully, then exit.
			time.Sleep(time.Second * 5)
			os.Exit(0)
		case <-ctx.Done():
			fmt.Println("shutting down on context done")
		}
	}()

	rawlog, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to create logger: %+v\n", err)
	}
	defer func() {
		log.Printf("main function teardown\n")
		err := rawlog.Sync()
		if err != nil {
			log.Printf("failed to sync logger on teardown: %+v", err.Error())
		}
	}()

	log := rawlog.Sugar().With("source", "supercollider_main")

	log.Info("Initializing PLC...")
	memPLC, err := NewPLC(ctx)
	if err != nil {
		log.Fatalf("failed to create plc: %+v\n", err)
	}

	log.Info("Initializing in-memory PDS...")
	tp, err := SetupInMemoryPDS(ctx, ".supercollider", memPLC)
	if err != nil {
		log.Fatalf("failed to setup pds: %+v\n", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	log.Infof("starting pds on %s ...", tp.Listener.Addr().String())
	if err := RunPDS(ctx, tp); err != nil {
		log.Fatalf("failed to run pds: %+v\n", err)
	}
	log.Info("...started pds")

	concurrency := 10
	numUsers := 1000

	log.Infof("creating %d users...", numUsers)
	users, errors := CreateUsersParallel(ctx, tp, concurrency, numUsers)
	if len(errors) > 0 {
		log.Errorf("failed to create some users: %+v\n", errors)
	}
	log.Infof("...created %d users", len(users))

	wg.Wait()
}

type Result struct {
	User  *testing.TestUser
	Error error
}

func CreateUsersParallel(ctx context.Context, tp *testing.TestPDS, concurrency int, numUsers int) ([]*testing.TestUser, []error) {
	users := make([]*testing.TestUser, 0)
	handles := supercollider.GenHandles("supercollider", numUsers)

	workCh := make(chan int, numUsers)
	resultsCh := make(chan Result, numUsers)
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range workCh {
				user, err := tp.NewUser(ctx, handles[idx])
				resultsCh <- Result{User: user, Error: err}
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
		}()
	}

	for i := 0; i < numUsers; i++ {
		workCh <- i
	}

	close(workCh)

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var errors []error
	for result := range resultsCh {
		if result.Error != nil {
			errors = append(errors, result.Error)
			continue
		}
		users = append(users, result.User)
	}

	return users, errors
}

func RunPDS(ctx context.Context, tp *testing.TestPDS) error {
	go func() {
		if err := tp.Server.RunAPIWithListener(tp.Listener); err != nil {
			fmt.Println(err)
		}
	}()
	time.Sleep(time.Millisecond * 10)

	tp.Shutdown = func() {
		tp.Server.Shutdown(context.TODO())
	}

	return nil
}

func NewPLC(ctx context.Context) (*plc.FakeDid, error) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&mode=rwc"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	rawDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	rawDB.SetMaxOpenConns(1)
	return plc.NewFakeDid(db), nil
}

func SetupInMemoryPDS(ctx context.Context, suffix string, plc plc.PLCClient) (*testing.TestPDS, error) {
	dir, err := os.MkdirTemp("", "supercollider")
	if err != nil {
		return nil, err
	}

	maindb, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&mode=rwc"))
	if err != nil {
		return nil, err
	}

	rawDB, err := maindb.DB()
	if err != nil {
		return nil, err
	}
	rawDB.SetMaxOpenConns(1)

	tx := maindb.Exec("PRAGMA journal_mode=WAL;")
	if tx.Error != nil {
		return nil, tx.Error
	}

	cardb, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&mode=rwc"))
	if err != nil {
		return nil, err
	}

	rawDB, err = cardb.DB()
	if err != nil {
		return nil, err
	}
	rawDB.SetMaxOpenConns(1)

	cspath := filepath.Join(dir, "carstore")
	if err := os.Mkdir(cspath, 0775); err != nil {
		return nil, err
	}

	cs, err := carstore.NewCarStore(cardb, cspath)
	if err != nil {
		return nil, err
	}

	raw, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new ECDSA private key: %s", err)
	}
	serkey := &did.PrivKey{
		Raw:  raw,
		Type: did.KeyTypeP256,
	}

	var lc net.ListenConfig
	li, err := lc.Listen(ctx, "tcp", "localhost:0")
	if err != nil {
		return nil, err
	}

	host := li.Addr().String()
	srv, err := pds.NewServer(maindb, cs, serkey, suffix, host, plc, []byte(host+suffix))
	if err != nil {
		return nil, err
	}

	return testing.NewTestPDS(dir, srv, li), nil
}
