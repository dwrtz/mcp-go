package transport

import (
	"context"
	"sync"

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

	// GetRouter returns the message router
	GetRouter() *MessageRouter

	// Close shuts down the transport
	Close() error

	// Done returns a channel that is closed when the transport is closed
	Done() <-chan struct{}

	// Logf logs a formatted message
	Logf(format string, args ...interface{})
}

type Logger interface {
	// Logf prints a formatted log message
	Logf(format string, args ...interface{})
}

// MessageRouter handles routing of messages to appropriate channels
type MessageRouter struct {
	// Channels for incoming messages
	Requests      chan *types.Message
	Responses     chan *types.Message
	Notifications chan *types.Message
	Errors        chan error

	// Control channels
	done chan struct{}
	once sync.Once

	logger Logger
}

const defaultChannelSize = 10

func NewMessageRouter(logger Logger) *MessageRouter {
	return &MessageRouter{
		Requests:      make(chan *types.Message, defaultChannelSize),
		Responses:     make(chan *types.Message, defaultChannelSize),
		Notifications: make(chan *types.Message, defaultChannelSize),
		Errors:        make(chan error, defaultChannelSize),
		done:          make(chan struct{}),
		logger:        logger,
	}
}

// Handle implements MessageHandler.Handle
func (r *MessageRouter) Handle(ctx context.Context, msg *types.Message) {
	if msg == nil {
		r.logger.Logf("Received nil message")
		return
	}

	if err := msg.Validate(); err != nil {
		r.logger.Logf("Invalid message: %v", err)
		return
	}

	// Route based on message type
	select {
	case <-r.done:
		r.logger.Logf("Router closed, dropping message")
		return
	case <-ctx.Done():
		r.logger.Logf("Context cancelled while routing message")
		return
	default:
		if msg.Method == "" {
			// This is a response
			select {
			case r.Responses <- msg:
			default:
				r.logger.Logf("Response channel full, dropping message")
			}
		} else if msg.ID == nil {
			// This is a notification
			select {
			case r.Notifications <- msg:
			default:
				r.logger.Logf("Notification channel full, dropping message")
			}
		} else {
			// This is a request
			select {
			case r.Requests <- msg:
			default:
				r.logger.Logf("Request channel full, dropping message")
			}
		}
	}
}

// Done returns a channel that's closed when the router is shutting down
func (r *MessageRouter) Done() <-chan struct{} {
	return r.done
}

// Close closes the router and its channels
func (r *MessageRouter) Close() {
	r.once.Do(func() {
		close(r.done)
		close(r.Requests)
		close(r.Responses)
		close(r.Notifications)
		close(r.Errors)
	})
}
