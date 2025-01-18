package mock

import (
	"context"
	"sync"

	"github.com/dwrtz/mcp-go/internal/testutil"
	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// MockTransport implements Transport for testing
type MockTransport struct {
	router *transport.MessageRouter
	logger *testutil.TestLogger
	done   chan struct{}

	// For inspecting what was sent
	sentMessages []types.Message
	mu           sync.Mutex
}

// NewMockTransport creates a new mock transport
func NewMockTransport(logger *testutil.TestLogger) *MockTransport {
	return &MockTransport{
		router:       transport.NewMessageRouter(logger),
		logger:       logger,
		done:         make(chan struct{}),
		sentMessages: make([]types.Message, 0),
	}
}

func (t *MockTransport) Start(ctx context.Context) error {
	// Nothing to do for mock
	return nil
}

func (t *MockTransport) Send(ctx context.Context, msg *types.Message) error {
	t.mu.Lock()
	t.sentMessages = append(t.sentMessages, *msg)
	t.mu.Unlock()

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

// Test helper methods

// GetSentMessages returns all messages that were sent through this transport
func (t *MockTransport) GetSentMessages() []types.Message {
	t.mu.Lock()
	defer t.mu.Unlock()

	msgs := make([]types.Message, len(t.sentMessages))
	copy(msgs, t.sentMessages)
	return msgs
}

// ClearSentMessages clears the history of sent messages
func (t *MockTransport) ClearSentMessages() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sentMessages = make([]types.Message, 0)
}

// SimulateReceive simulates receiving a message from the other end
func (t *MockTransport) SimulateReceive(ctx context.Context, msg *types.Message) {
	t.router.Handle(ctx, msg)
}
