package server

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// RootsServer provides server-side roots functionality
type RootsServer struct {
	base  *base.Server
	mu    sync.RWMutex
	roots []types.Root
}

// NewRootsServer creates a new RootsServer
func NewRootsServer(base *base.Server) *RootsServer {
	s := &RootsServer{
		base:  base,
		roots: make([]types.Root, 0),
	}

	// Register request handler for roots/list
	base.RegisterRequestHandler(methods.ListRoots, s.handleListRoots)

	return s
}

// SetRoots updates the list of available roots
func (s *RootsServer) SetRoots(ctx context.Context, roots []types.Root) error {
	// Validate all roots first
	for _, root := range roots {
		if err := root.Validate(); err != nil {
			return err
		}
	}

	s.mu.Lock()
	s.roots = roots
	s.mu.Unlock()

	// Notify clients of the change
	return s.base.SendNotification(ctx, methods.RootsChanged, nil)
}

// handleListRoots handles the roots/list request
func (s *RootsServer) handleListRoots(ctx context.Context, params json.RawMessage) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &types.ListRootsResult{
		Roots: s.roots,
	}, nil
}
