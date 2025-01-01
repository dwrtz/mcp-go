package transport

import (
	"context"
	"time"

	"github.com/dwrtz/mcp-go/pkg/types"
	"github.com/sourcegraph/jsonrpc2"
)

// Handler handles incoming messages from a Transport
type Handler interface {
	// Handle processes an incoming message and optionally returns a response
	Handle(ctx context.Context, msg *types.Message) (*types.Message, error)
}

// Transport represents a bidirectional communication channel for MCP messages
type Transport interface {
	// Start begins listening for messages. It blocks until the context is cancelled
	// or an error occurs.
	Start(ctx context.Context) error

	// Send sends a message through the transport
	Send(ctx context.Context, msg *types.Message) error

	// Close closes the transport and cleans up any resources
	Close() error

	// SetHandler sets the handler for incoming messages
	SetHandler(handler Handler)

	// Done returns a channel that is closed when this transport is closed.
	Done() <-chan struct{}
}

// Options contains configuration options for transports
type Options struct {
	// Handler is the handler for incoming messages
	Handler Handler

	// BufferSize is the size of message buffers (if applicable)
	BufferSize int

	// Timeout is the default timeout for operations
	Timeout time.Duration
}

// BaseTransport provides common functionality for transport implementations
type BaseTransport struct {
	handler Handler
	Conn    *jsonrpc2.Conn
	done    chan struct{}
}

// NewBaseTransport creates a new BaseTransport
func NewBaseTransport() *BaseTransport {
	return &BaseTransport{
		done: make(chan struct{}),
	}
}

// Done implements Transport
func (t *BaseTransport) Done() <-chan struct{} {
	return t.done
}

// SetHandler implements Transport
func (t *BaseTransport) SetHandler(h Handler) {
	t.handler = h
}

// Close implements Transport
func (t *BaseTransport) Close() error {
	select {
	case <-t.done:
		return nil
	default:
		close(t.done)
		if t.Conn != nil {
			return t.Conn.Close()
		}
		return nil
	}
}

// IsClosed returns whether the transport has been closed
func (t *BaseTransport) IsClosed() bool {
	select {
	case <-t.done:
		return true
	default:
		return false
	}
}

// Handle processes incoming JSON-RPC messages and routes them to the handler
func (t *BaseTransport) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
	if t.handler == nil {
		return nil, &jsonrpc2.Error{Code: types.MethodNotFound, Message: "no handler registered"}
	}

	// Create a copy of the ID to preserve it exactly
	id := req.ID

	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      &id, // Use the exact ID from the request
		Method:  req.Method,
		Params:  req.Params,
	}

	resp, err := t.handler.Handle(ctx, msg)
	if err != nil {
		if rpcErr, ok := err.(*jsonrpc2.Error); ok {
			return nil, rpcErr
		}
		return nil, &jsonrpc2.Error{Code: types.InternalError, Message: err.Error()}
	}

	return resp, nil
}
