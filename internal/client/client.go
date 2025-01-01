package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// clientHandler receives all responses from the server.
type clientHandler struct {
	client *Client
	pingID types.ID
	logger transport.Logger
	once   sync.Once
}

func (h *clientHandler) Handle(ctx context.Context, msg *types.Message) (*types.Message, error) {
	if h.logger != nil {
		h.logger.Logf("Client received message: %+v", msg)
	}

	// Check if it's a response to our ping
	if msg.ID != nil {
		if msg.Result != nil && *msg.ID == h.pingID {
			if h.logger != nil {
				h.logger.Logf("Client received matching ping response, closing")
			}
			h.once.Do(func() {
				h.client.Close()
			})
		}
	}
	return nil, nil
}

// Client is a minimal struct that can send requests and handle responses.
type Client struct {
	transport transport.Transport
	ready     chan struct{}
	logger    transport.Logger
	mu        sync.Mutex
	closed    bool
}

// NewClient creates a new Client with the given transport and optional logger
func NewClient(t transport.Transport) *Client {
	return &Client{
		transport: t,
		ready:     make(chan struct{}),
	}
}

func (c *Client) SetLogger(l transport.Logger) {
	if l == nil {
		l = transport.NoopLogger{}
	}
	c.logger = l
}

func (c *Client) Start(ctx context.Context, pingID types.ID) error {
	h := &clientHandler{
		client: c,
		pingID: pingID,
		logger: c.logger,
	}
	c.transport.SetHandler(h)

	go func() {
		err := c.transport.Start(ctx)
		if err != nil && c.logger != nil {
			c.logger.Logf("Client transport stopped: %v", err)
		}
	}()

	close(c.ready)
	return nil
}

// Ping sends a "ping" request to the server.
func (c *Client) Ping(ctx context.Context, id types.ID) error {
	select {
	case <-c.ready:
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for transport: %w", ctx.Err())
	}

	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      &id,
		Method:  "ping",
	}

	if c.logger != nil {
		c.logger.Logf("Client sending ping with ID %v", id)
	}

	err := c.transport.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send ping: %w", err)
	}
	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true
	return c.transport.Close()
}

func (c *Client) Done() <-chan struct{} {
	return c.transport.Done()
}
