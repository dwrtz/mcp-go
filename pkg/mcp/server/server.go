package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/internal/server/prompts"
	"github.com/dwrtz/mcp-go/internal/server/resources"
	"github.com/dwrtz/mcp-go/internal/server/roots"
	"github.com/dwrtz/mcp-go/internal/server/sampling"
	"github.com/dwrtz/mcp-go/internal/server/tools"
	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/internal/transport/stdio"
	"github.com/dwrtz/mcp-go/pkg/logger"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// NewDefaultServer creates an MCP server with default settings
func NewDefaultServer(opts ...ServerOption) *Server {

	// Create transport
	t := stdio.NewTransport(os.Stdin, os.Stdout)

	// Create server
	s := NewServer(t, opts...)

	return s
}

// Server represents a Model Context Protocol server
type Server struct {
	base *base.Base

	// Feature-specific servers
	roots     *roots.Server
	resources *resources.Server
	prompts   *prompts.Server
	tools     *tools.Server
	sampling  *sampling.Server

	// Server capabilities
	capabilities types.ServerCapabilities

	// Server info
	info types.Implementation
}

// ServerOption is a function that configures a Server
type ServerOption func(*Server)

// WithLogger sets the logger for the server
func WithLogger(l logger.Logger) ServerOption {
	return func(s *Server) {
		s.base.SetLogger(l)
	}
}

// WithResources enables resources functionality on the server
func WithResources(initialResources []types.Resource, initialTemplates []types.ResourceTemplate) ServerOption {
	return func(s *Server) {
		s.capabilities.Resources = &types.ResourcesServerCapabilities{
			Subscribe:   true,
			ListChanged: true,
		}
		s.resources = resources.NewServer(s.base, initialResources, initialTemplates)
	}
}

// WithPrompts enables prompts functionality on the server
func WithPrompts(initialPrompts []types.Prompt) ServerOption {
	return func(s *Server) {
		s.capabilities.Prompts = &types.PromptsServerCapabilities{
			ListChanged: true,
		}
		s.prompts = prompts.NewServer(s.base, initialPrompts)
	}
}

// WithTools enables tools functionality on the server
func WithTools(initialTools ...types.McpTool) ServerOption {
	return func(s *Server) {
		s.capabilities.Tools = &types.ToolsServerCapabilities{
			ListChanged: true,
		}
		s.tools = tools.NewServer(s.base, initialTools)
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

// Start begins processing messages but also makes sure that the server's ctx
// is canceled if the transport closes, so you can shut down everything automatically.
func (s *Server) Start(ctx context.Context) error {
	// Create a child context we can cancel if the transport closes:
	serverCtx, cancelFunc := context.WithCancel(ctx)

	// Start the underlying base (which spins up its own goroutine)
	if err := s.base.Start(serverCtx); err != nil {
		cancelFunc()
		return fmt.Errorf("failed to start base transport: %w", err)
	}

	// Watch for transport closure. When that happens, we cancel serverCtx.
	go func() {
		<-s.base.GetRouter().Done() // transport closed
		s.Close()
		cancelFunc()
	}()

	// We return immediately; background goroutines handle the requests.
	return nil
}

// Close shuts down the server
func (s *Server) Close() error {
	return s.base.Close()
}

// Done returns a channel that is closed when the transport is closed
func (s *Server) Done() <-chan struct{} {
	return s.base.Done()
}

// SupportsRoots returns whether the client supports roots functionality
func (s *Server) SupportsRoots() bool {
	return s.roots != nil
}

// SupportsResources returns whether the server supports resources functionality
func (s *Server) SupportsResources() bool {
	return s.resources != nil
}

// SupportsPrompts returns whether the server supports prompts functionality
func (s *Server) SupportsPrompts() bool {
	return s.prompts != nil
}

// SupportsTools returns whether the server supports tools functionality
func (s *Server) SupportsTools() bool {
	return s.tools != nil
}

// SupportsSampling returns whether the client supports sampling functionality
func (s *Server) SupportsSampling() bool {
	return s.sampling != nil
}

// handleInitialize handles the initialize request from clients
func (s *Server) handleInitialize(ctx context.Context, params *json.RawMessage) (interface{}, error) {
	if params == nil {
		return nil, types.NewError(types.InvalidParams, "missing params")
	}

	var req types.InitializeRequest
	if err := json.Unmarshal(*params, &req); err != nil {
		return nil, fmt.Errorf("failed to parse initialize request: %w", err)
	}

	// Verify protocol version compatibility
	if req.ProtocolVersion != types.LatestProtocolVersion {
		return nil, fmt.Errorf("client protocol version %s not supported", req.ProtocolVersion)
	}

	// Initialize roots and sampling server if client supports it
	if req.Capabilities.Roots != nil {
		s.roots = roots.NewServer(s.base)
		s.OnRootsChanged(func() {
			// default noop
			s.base.Logf("from client: %s", methods.RootsChanged)
		})
	}

	if req.Capabilities.Sampling != nil {
		s.sampling = sampling.NewServer(s.base)
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

// Resource Methods

// SetResources updates the list of available resources and notifies connected clients.
// Returns an error if resources are not supported or if the update fails.
func (s *Server) SetResources(ctx context.Context, resources []types.Resource) error {
	if !s.SupportsResources() {
		return types.NewError(types.MethodNotFound, "resources not supported")
	}
	return s.resources.SetResources(ctx, resources)
}

// SetResourceTemplates updates the list of available resource templates.
func (s *Server) SetResourceTemplates(ctx context.Context, templates []types.ResourceTemplate) {
	if s.SupportsResources() {
		s.resources.SetTemplates(ctx, templates)
	}
}

// RegisterContentHandler registers a handler for reading resource contents.
// The handler is called when clients request to read resources with URIs matching the given prefix.
func (s *Server) RegisterContentHandler(uriPrefix string, handler resources.ContentHandler) {
	if s.SupportsResources() {
		s.resources.RegisterContentHandler(uriPrefix, handler)
	}
}

// NotifyResourceUpdated notifies subscribed clients that a resource has changed.
// Returns an error if resources are not supported or if notification fails.
func (s *Server) NotifyResourceUpdated(ctx context.Context, uri string) error {
	if !s.SupportsResources() {
		return types.NewError(types.MethodNotFound, "resources not supported")
	}
	return s.resources.NotifyResourceUpdated(ctx, uri)
}

// Prompt Methods

// SetPrompts updates the list of available prompts and notifies connected clients.
// Returns an error if prompts are not supported or if the update fails.
func (s *Server) SetPrompts(ctx context.Context, prompts []types.Prompt) error {
	if !s.SupportsPrompts() {
		return types.NewError(types.MethodNotFound, "prompts not supported")
	}
	return s.prompts.SetPrompts(ctx, prompts)
}

// RegisterPromptGetter registers a handler for retrieving prompt contents.
// The handler is called when clients request prompts by the given name.
func (s *Server) RegisterPromptGetter(name string, getter prompts.PromptGetter) {
	if s.SupportsPrompts() {
		s.prompts.RegisterPromptGetter(name, getter)
	}
}

// Tool Methods

// SetTools updates the list of available tools and notifies connected clients.
// Returns an error if tools are not supported or if the update fails.
func (s *Server) SetTools(ctx context.Context, newTools []types.McpTool) error {
	if !s.SupportsTools() {
		return types.NewError(types.MethodNotFound, "tools not supported")
	}
	return s.tools.SetTools(ctx, newTools)
}

// Root Methods

// List requests the list of available roots from the connected client.
// Returns an error if roots are not supported by the client.
func (s *Server) ListRoots(ctx context.Context) ([]types.Root, error) {
	if !s.SupportsRoots() {
		return nil, types.NewError(types.MethodNotFound, "roots not supported")
	}
	return s.roots.ListRoots(ctx)
}

// OnRootsChanged registers a callback for when the client's root list changes.
// The callback is not invoked if roots are not supported.
func (s *Server) OnRootsChanged(callback func()) {
	if s.SupportsRoots() {
		s.roots.OnRootsChanged(callback)
	}
}

// Sampling Methods

// CreateMessage requests a sample from the language model.
// Returns an error if sampling is not supported.
func (s *Server) CreateMessage(ctx context.Context, req *types.CreateMessageRequest) (*types.CreateMessageResult, error) {
	if !s.SupportsSampling() {
		return nil, types.NewError(types.MethodNotFound, "sampling not supported")
	}
	return s.sampling.CreateMessage(ctx, req)
}
