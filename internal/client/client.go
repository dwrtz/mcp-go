package client

import (
	"context"
	"sync"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
)

type Client struct {
	transport transport.Transport
	handlers  map[string]transport.MessageHandlerFunc
	mu        sync.RWMutex
	logger    transport.Logger
}

func New(t transport.Transport, logger transport.Logger) *Client {
	c := &Client{
		transport: t,
		handlers:  make(map[string]transport.MessageHandlerFunc),
		logger:    logger,
	}

	// Set ourselves as the transport's message handler
	t.SetHandler(c)

	return c
}

// Handle implements transport.MessageHandler
func (c *Client) Handle(ctx context.Context, msg *types.Message) (*types.Message, error) {
	c.mu.RLock()
	handler, ok := c.handlers[msg.Method]
	c.mu.RUnlock()

	if !ok {
		return nil, types.NewError(types.MethodNotFound, "no handler registered for method: "+msg.Method)
	}

	return handler(ctx, msg)
}

func (c *Client) RegisterHandler(method string, handler transport.MessageHandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[method] = handler
}

func (c *Client) Send(ctx context.Context, msg *types.Message) error {
	return c.transport.Send(ctx, msg)
}

func (c *Client) Start(ctx context.Context) error {
	return c.transport.Start(ctx)
}

func (c *Client) Stop() error {
	return c.transport.Close()
}
