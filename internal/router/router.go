package router

import (
	"context"
	"sync"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// HandlerFunc is a function that handles a specific method
type HandlerFunc func(ctx context.Context, msg *types.Message) (*types.Message, error)

// Router implements transport.Handler and routes messages to registered handlers
type Router struct {
	handlers map[string]HandlerFunc
	mu       sync.RWMutex
	logger   transport.Logger
}

// Option is a function that configures a Router
type Option func(*Router)

// WithLogger sets the logger for the router
func WithLogger(logger transport.Logger) Option {
	return func(r *Router) {
		if logger == nil {
			logger = transport.NoopLogger{}
		}
		r.logger = logger
	}
}

// New creates a new Router
func New(opts ...Option) *Router {
	r := &Router{
		handlers: make(map[string]HandlerFunc),
		logger:   transport.NoopLogger{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// RegisterHandler adds a new method handler
func (r *Router) RegisterHandler(method string, handler HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[method] = handler
	if r.logger != nil {
		r.logger.Logf("Registered handler for method: %s", method)
	}
}

// UnregisterHandler removes a method handler
func (r *Router) UnregisterHandler(method string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, method)
	if r.logger != nil {
		r.logger.Logf("Unregistered handler for method: %s", method)
	}
}

// Handle implements transport.Handler
func (r *Router) Handle(ctx context.Context, msg *types.Message) (*types.Message, error) {
	r.mu.RLock()
	handler, ok := r.handlers[msg.Method]
	r.mu.RUnlock()

	if !ok {
		if r.logger != nil {
			r.logger.Logf("No handler registered for method: %s", msg.Method)
		}
		return nil, types.NewError(
			types.MethodNotFound,
			"no handler registered for method: "+msg.Method,
		)
	}

	if r.logger != nil {
		r.logger.Logf("Routing message: method=%s id=%v", msg.Method, msg.ID)
	}

	return handler(ctx, msg)
}
