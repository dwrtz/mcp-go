package mock

import (
	"context"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// MockTransport implements transport.Transport for testing
type MockTransport struct {
	router *transport.MessageRouter
	logger transport.Logger
	done   chan struct{}
}

// NewMockTransport creates a new mock transport
func NewMockTransport(logger transport.Logger) *MockTransport {
	return &MockTransport{
		router: transport.NewMessageRouter(logger),
		logger: logger,
		done:   make(chan struct{}),
	}
}

func (t *MockTransport) Start(ctx context.Context) error {
	return nil // Nothing to do for mock
}

func (t *MockTransport) Send(ctx context.Context, msg *types.Message) error {
	// Route the message through our router
	t.router.Handle(ctx, msg)
	return nil
}

func (t *MockTransport) GetRouter() *transport.MessageRouter {
	return t.router
}

func (t *MockTransport) Close() error {
	select {
	case <-t.done:
		return nil
	default:
		close(t.done)
		t.router.Close()
		return nil
	}
}

func (t *MockTransport) Done() <-chan struct{} {
	return t.done
}

func (t *MockTransport) Logf(format string, args ...interface{}) {
	t.logger.Logf(format, args...)
}
