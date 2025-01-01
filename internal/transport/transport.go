package transport

import (
	"context"

	"github.com/dwrtz/mcp-go/pkg/types"
)

// MessageHandler handles incoming MCP messages by routing them to appropriate channels
type MessageHandler interface {
	// Handle processes an incoming message
	Handle(ctx context.Context, msg *types.Message)
}

// Transport defines the interface for MCP message transport
type Transport interface {
	// Start begins listening for messages
	Start(ctx context.Context) error

	// Send sends a message through the transport
	Send(ctx context.Context, msg *types.Message) error

	// SetHandler sets the handler for incoming messages
	SetHandler(handler MessageHandler)

	// Close shuts down the transport
	Close() error

	// Done returns a channel that is closed when the transport is closed
	Done() <-chan struct{}
}

// Logger defines a minimal interface for logging
type Logger interface {
	// Logf prints a formatted log message
	Logf(format string, args ...interface{})
}
