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

// RootsClient provides client-side roots functionality
type RootsClient struct {
	base  *base.Base
	mu    sync.RWMutex
	roots []types.Root
}

// NewRootsClient creates a new RootsClient
func NewRootsClient(base *base.Base, initialRoots []types.Root) *RootsClient {
	c := &RootsClient{
		base:  base,
		roots: initialRoots,
	}
	base.RegisterRequestHandler(methods.ListRoots, c.handleListRoots)
	return c
}

func (c *RootsClient) SetRoots(ctx context.Context, roots []types.Root) error {
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
func (c *RootsClient) handleListRoots(ctx context.Context, params *json.RawMessage) (interface{}, error) {
	return &types.ListRootsResult{
		Roots: c.roots,
	}, nil
}
