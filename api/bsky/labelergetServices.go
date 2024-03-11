// Code generated by cmd/lexgen (see Makefile's lexgen); DO NOT EDIT.

package bsky

// schema: app.bsky.labeler.getServices

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
)

// LabelerGetServices_Output is the output of a app.bsky.labeler.getServices call.
type LabelerGetServices_Output struct {
	Views []*LabelerGetServices_Output_Views_Elem `json:"views" cborgen:"views"`
}

type LabelerGetServices_Output_Views_Elem struct {
	LabelerDefs_LabelerView         *LabelerDefs_LabelerView
	LabelerDefs_LabelerViewDetailed *LabelerDefs_LabelerViewDetailed
}

func (t *LabelerGetServices_Output_Views_Elem) MarshalJSON() ([]byte, error) {
	if t.LabelerDefs_LabelerView != nil {
		t.LabelerDefs_LabelerView.LexiconTypeID = "app.bsky.labeler.defs#labelerView"
		return json.Marshal(t.LabelerDefs_LabelerView)
	}
	if t.LabelerDefs_LabelerViewDetailed != nil {
		t.LabelerDefs_LabelerViewDetailed.LexiconTypeID = "app.bsky.labeler.defs#labelerViewDetailed"
		return json.Marshal(t.LabelerDefs_LabelerViewDetailed)
	}
	return nil, fmt.Errorf("cannot marshal empty enum")
}
func (t *LabelerGetServices_Output_Views_Elem) UnmarshalJSON(b []byte) error {
	typ, err := util.TypeExtract(b)
	if err != nil {
		return err
	}

	switch typ {
	case "app.bsky.labeler.defs#labelerView":
		t.LabelerDefs_LabelerView = new(LabelerDefs_LabelerView)
		return json.Unmarshal(b, t.LabelerDefs_LabelerView)
	case "app.bsky.labeler.defs#labelerViewDetailed":
		t.LabelerDefs_LabelerViewDetailed = new(LabelerDefs_LabelerViewDetailed)
		return json.Unmarshal(b, t.LabelerDefs_LabelerViewDetailed)

	default:
		return nil
	}
}

// LabelerGetServices calls the XRPC method "app.bsky.labeler.getServices".
func LabelerGetServices(ctx context.Context, c *xrpc.Client, detailed bool, dids []string) (*LabelerGetServices_Output, error) {
	var out LabelerGetServices_Output

	params := map[string]interface{}{
		"detailed": detailed,
		"dids":     dids,
	}
	if err := c.Do(ctx, xrpc.Query, "", "app.bsky.labeler.getServices", params, nil, &out); err != nil {
		return nil, err
	}

	return &out, nil
}