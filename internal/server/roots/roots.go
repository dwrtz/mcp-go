package roots

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// Server provides server-side roots functionality
type Server struct {
	base *base.Base
}

// NewServer creates a new Server
func NewServer(base *base.Base) *Server {
	return &Server{base: base}
}

// ListRoots requests the list of available roots from the client
func (s *Server) ListRoots(ctx context.Context) ([]types.Root, error) {
	req := &types.ListRootsRequest{
		Method: methods.ListRoots,
	}

	resp, err := s.base.SendRequest(ctx, methods.ListRoots, req)
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
func (s *Server) OnRootsChanged(callback func()) {
	s.base.RegisterNotificationHandler(methods.RootsChanged, func(ctx context.Context, params json.RawMessage) {
		callback()
	})
}
