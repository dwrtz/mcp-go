package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dwrtz/mcp-go/internal/router"
	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// Server handles incoming client connections and routes messages.
type Server struct {
	transport transport.Transport
	router    *router.Router
	done      chan struct{}
	logger    transport.Logger
	mu        sync.Mutex
	closed    bool
}

// Option is a function that configures a Server
type Option func(*Server)

// WithLogger sets the logger for the server
func WithLogger(logger transport.Logger) Option {
	return func(s *Server) {
		if logger == nil {
			logger = transport.NoopLogger{}
		}
		s.logger = logger
		s.router = router.New(router.WithLogger(logger))
	}
}

// NewServer creates a new Server with the given transport and options
func NewServer(t transport.Transport, opts ...Option) *Server {
	s := &Server{
		transport: t,
		done:      make(chan struct{}),
		logger:    transport.NoopLogger{},
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Create router if not created by options
	if s.router == nil {
		s.router = router.New()
	}

	// Set router as transport handler
	t.SetHandler(s.router)

	return s
}

// RegisterHandler registers a handler for a method
func (s *Server) RegisterHandler(method string, handler router.HandlerFunc) {
	s.router.RegisterHandler(method, handler)
}

// UnregisterHandler removes a handler for a method
func (s *Server) UnregisterHandler(method string) {
	s.router.UnregisterHandler(method)
}

// Start begins listening for client connections
func (s *Server) Start(ctx context.Context) error {
	if s.logger != nil {
		s.logger.Logf("Server starting...")
	}

	go func() {
		err := s.transport.Start(ctx)
		if err != nil && s.logger != nil {
			s.logger.Logf("Server transport stopped: %v", err)
		}
	}()

	return nil
}

// Send sends a message through the transport
func (s *Server) Send(ctx context.Context, method string, id types.ID, params interface{}) error {
	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      &id,
		Method:  method,
	}

	if params != nil {
		// Convert params to json.RawMessage if provided
		raw, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params: %w", err)
		}
		rawMsg := json.RawMessage(raw)
		msg.Params = &rawMsg
	}

	if s.logger != nil {
		s.logger.Logf("Server sending message: method=%s id=%v", method, id)
	}

	err := s.transport.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}

// Close shuts down the server and cleans up resources
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true
	close(s.done)
	return s.transport.Close()
}

func (s *Server) Done() <-chan struct{} {
	return s.transport.Done()
}
