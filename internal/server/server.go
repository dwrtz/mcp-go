package server

import (
	"context"
	"sync"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
)

type Server struct {
	transport transport.Transport
	handlers  map[string]transport.MessageHandlerFunc
	mu        sync.RWMutex
	logger    transport.Logger
}

func New(t transport.Transport, logger transport.Logger) *Server {
	s := &Server{
		transport: t,
		handlers:  make(map[string]transport.MessageHandlerFunc),
		logger:    logger,
	}

	// Set ourselves as the transport's message handler
	t.SetHandler(s)

	return s
}

// Handle implements transport.MessageHandler
func (s *Server) Handle(ctx context.Context, msg *types.Message) (*types.Message, error) {
	s.mu.RLock()
	handler, ok := s.handlers[msg.Method]
	s.mu.RUnlock()

	if !ok {
		return nil, types.NewError(types.MethodNotFound, "no handler registered for method: "+msg.Method)
	}

	return handler(ctx, msg)
}

func (s *Server) RegisterHandler(method string, handler transport.MessageHandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = handler
}

func (s *Server) Send(ctx context.Context, msg *types.Message) error {
	return s.transport.Send(ctx, msg)
}

func (s *Server) Start(ctx context.Context) error {
	return s.transport.Start(ctx)
}

func (s *Server) Stop() error {
	return s.transport.Close()
}
