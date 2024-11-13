// Code generated by cmd/lexgen (see Makefile's lexgen); DO NOT EDIT.

package ozone

// schema: tools.ozone.set.addValues

import (
	"context"

	"github.com/bluesky-social/indigo/xrpc"
)

// SetAddValues_Input is the input argument to a tools.ozone.set.addValues call.
type SetAddValues_Input struct {
	// name: Name of the set to add values to
	Name string `json:"name" cborgen:"name"`
	// values: Array of string values to add to the set
	Values []string `json:"values" cborgen:"values"`
}

// SetAddValues calls the XRPC method "tools.ozone.set.addValues".
func SetAddValues(ctx context.Context, c *xrpc.Client, input *SetAddValues_Input) error {
	if err := c.Do(ctx, xrpc.Procedure, "application/json", "tools.ozone.set.addValues", nil, input, nil); err != nil {
		return err
	}

	return nil
}