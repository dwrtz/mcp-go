package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/server/prompts"
	"github.com/dwrtz/mcp-go/pkg/server/resources"
	"github.com/dwrtz/mcp-go/pkg/server/roots"
	"github.com/dwrtz/mcp-go/pkg/server/sampling"
	"github.com/dwrtz/mcp-go/pkg/server/tools"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// Server represents a Model Context Protocol server
type Server struct {
	base *base.Base

	// Feature-specific servers
	roots     *roots.RootsServer
	resources *resources.ResourcesServer
	prompts   *prompts.PromptsServer
	tools     *tools.ToolsServer
	sampling  *sampling.SamplingServer

	// Server capabilities
	capabilities types.ServerCapabilities

	// Server info
	info types.Implementation
}

// ServerOption is a function that configures a Server
type ServerOption func(*Server)

// WithResources enables resources functionality on the server
func WithResources(initialResources []types.Resource, initialTemplates []types.ResourceTemplate) ServerOption {
	return func(s *Server) {
		s.capabilities.Resources = &types.ResourcesServerCapabilities{
			Subscribe:   true,
			ListChanged: true,
		}
		s.resources = resources.NewResourcesServer(s.base, initialResources, initialTemplates)
	}
}

// WithPrompts enables prompts functionality on the server
func WithPrompts(initialPrompts []types.Prompt) ServerOption {
	return func(s *Server) {
		s.capabilities.Prompts = &types.PromptsServerCapabilities{
			ListChanged: true,
		}
		s.prompts = prompts.NewPromptsServer(s.base, initialPrompts)
	}
}

// WithTools enables tools functionality on the server
func WithTools(initialTools []types.Tool) ServerOption {
	return func(s *Server) {
		s.capabilities.Tools = &types.ToolsServerCapabilities{
			ListChanged: true,
		}
		s.tools = tools.NewToolsServer(s.base, initialTools)
	}
}

// NewServer creates a new MCP server
func NewServer(transport transport.Transport, opts ...ServerOption) *Server {
	s := &Server{
		base: base.NewBase(transport),
		info: types.Implementation{
			Name:    "mcp-go",
			Version: "0.1.0",
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
func (s *Server) Roots() *roots.RootsServer {
	return s.roots
}

// Resources returns the resources server if enabled
func (s *Server) Resources() *resources.ResourcesServer {
	return s.resources
}

// Prompts returns the prompts server if enabled
func (s *Server) Prompts() *prompts.PromptsServer {
	return s.prompts
}

// Tools returns the tools server if enabled
func (s *Server) Tools() *tools.ToolsServer {
	return s.tools
}

// Sampling returns the sampling server if enabled
func (s *Server) Sampling() *sampling.SamplingServer {
	return s.sampling
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

	// Initialize roots and sampling server if client supports it
	if req.Capabilities.Roots != nil {
		s.roots = roots.NewRootsServer(s.base)
	}

	if req.Capabilities.Sampling != nil {
		s.sampling = sampling.NewSamplingServer(s.base)
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
