package roots

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// RootsClient provides client-side roots functionality
type RootsClient struct {
	base *base.Base
}

// NewRootsClient creates a new RootsClient
func NewRootsClient(base *base.Base) *RootsClient {
	return &RootsClient{base: base}
}

// List requests the list of available roots from the server
func (c *RootsClient) List(ctx context.Context) ([]types.Root, error) {
	req := &types.ListRootsRequest{
		Method: methods.ListRoots,
	}

	resp, err := c.base.SendRequest(ctx, methods.ListRoots, req)
	if err != nil {
		return nil, err
	}

	// Check for error response
	if resp.Error != nil {
		return nil, resp.Error
	}

	// Parse response
	var result types.ListRootsResult
	if err := json.Unmarshal(*resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse roots list response: %w", err)
	}

	return result.Roots, nil
}

// OnRootsChanged registers a callback to be called when the roots list changes
func (c *RootsClient) OnRootsChanged(callback func()) {
	c.base.RegisterNotificationHandler(methods.RootsChanged, func(ctx context.Context, params json.RawMessage) {
		callback()
	})
}
