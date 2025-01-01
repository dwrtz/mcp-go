package message

import (
	"context"
	"sync"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
)

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

	logger transport.Logger
}

const defaultChannelSize = 10

func NewMessageRouter(logger transport.Logger) *MessageRouter {
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
