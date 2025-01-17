package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/server"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// Server represents a Model Context Protocol server
type Server struct {
	base *base.Server

	// Feature-specific servers
	roots *server.RootsServer

	// Server capabilities
	capabilities types.ServerCapabilities

	// Server info
	info types.Implementation
}

// ServerOption is a function that configures a Server
type ServerOption func(*Server)

// WithRoots enables roots functionality on the server
func WithRoots() ServerOption {
	return func(s *Server) {
		s.capabilities.Resources = &types.ResourcesServerCapabilities{
			Subscribe:   true,
			ListChanged: true,
		}
	}
}

// NewServer creates a new MCP server
func NewServer(transport transport.Transport, opts ...ServerOption) *Server {
	s := &Server{
		base: base.NewServer(transport),
		info: types.Implementation{
			Name:    "mcp-go",
			Version: "0.1.0", // TODO: Use version from build
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Register initialization handler
	s.base.RegisterRequestHandler(methods.Initialize, s.handleInitialize)
	s.base.RegisterNotificationHandler(methods.Initialized, s.handleInitialized)

	return s
}

// Start begins processing messages
func (s *Server) Start(ctx context.Context) error {
	return s.base.Start(ctx)
}

// Close shuts down the server
func (s *Server) Close() error {
	return s.base.Close()
}

// Roots returns the roots server if enabled
func (s *Server) Roots() *server.RootsServer {
	return s.roots
}

// handleInitialize handles the initialize request from clients
func (s *Server) handleInitialize(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req types.InitializeRequest
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("failed to parse initialize request: %w", err)
	}

	// Verify protocol version compatibility
	if req.ProtocolVersion != types.LatestProtocolVersion {
		return nil, fmt.Errorf("client protocol version %s not supported", req.ProtocolVersion)
	}

	// Initialize feature-specific servers based on capabilities
	if s.capabilities.Resources != nil {
		s.roots = server.NewRootsServer(s.base)
	}

	return &types.InitializeResult{
		ProtocolVersion: types.LatestProtocolVersion,
		Capabilities:    s.capabilities,
		ServerInfo:      s.info,
	}, nil
}

// handleInitialized handles the initialized notification from clients
func (s *Server) handleInitialized(ctx context.Context, params json.RawMessage) {
	// Nothing to do here, but we need to handle the notification
}
