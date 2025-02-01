package roots

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// Client provides client-side roots functionality
type Client struct {
	base  *base.Base
	mu    sync.RWMutex
	roots []types.Root
}

// NewClient creates a new Client
func NewClient(base *base.Base, initialRoots []types.Root) *Client {
	c := &Client{
		base:  base,
		roots: initialRoots,
	}
	base.RegisterRequestHandler(methods.ListRoots, c.handleListRoots)
	return c
}

// SetRoots sets the roots for the client
func (c *Client) SetRoots(ctx context.Context, roots []types.Root) error {
	// Validate all roots before setting
	for _, root := range roots {
		if err := root.Validate(); err != nil {
			return fmt.Errorf("invalid root %s: %w", root.URI, err)
		}
	}

	c.mu.Lock()
	c.roots = roots
	c.mu.Unlock()

	if c.base.Started {
		return c.base.SendNotification(ctx, methods.RootsChanged, nil)
	}
	return nil
}

// handleListRoots handles the roots/list request
func (c *Client) handleListRoots(ctx context.Context, params *json.RawMessage) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return &types.ListRootsResult{
		Roots: c.roots,
	}, nil
}
