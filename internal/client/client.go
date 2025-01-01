package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dwrtz/mcp-go/internal/router"
	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// Client is a minimal struct that can send requests and handle responses.
type Client struct {
	transport transport.Transport
	router    *router.Router
	ready     chan struct{}
	logger    transport.Logger
	mu        sync.Mutex
	closed    bool
}

// Option is a function that configures a Client
type Option func(*Client)

// WithLogger sets the logger for the client
func WithLogger(logger transport.Logger) Option {
	return func(c *Client) {
		if logger == nil {
			logger = transport.NoopLogger{}
		}
		c.logger = logger
		c.router = router.New(router.WithLogger(logger))
	}
}

// NewClient creates a new Client with the given transport and options
func NewClient(t transport.Transport, opts ...Option) *Client {
	c := &Client{
		transport: t,
		ready:     make(chan struct{}),
		logger:    transport.NoopLogger{},
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	// Create router if not created by options
	if c.router == nil {
		c.router = router.New()
	}

	// Set router as transport handler
	t.SetHandler(c.router)

	return c
}

// RegisterHandler registers a handler for a method
func (c *Client) RegisterHandler(method string, handler router.HandlerFunc) {
	c.router.RegisterHandler(method, handler)
}

// UnregisterHandler removes a handler for a method
func (c *Client) UnregisterHandler(method string) {
	c.router.UnregisterHandler(method)
}

// Start initiates the client connection
func (c *Client) Start(ctx context.Context) error {
	go func() {
		err := c.transport.Start(ctx)
		if err != nil && c.logger != nil {
			c.logger.Logf("Client transport stopped: %v", err)
		}
	}()

	close(c.ready)
	return nil
}

// Send sends a message through the transport
func (c *Client) Send(ctx context.Context, method string, id types.ID, params interface{}) error {
	select {
	case <-c.ready:
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for transport: %w", ctx.Err())
	}

	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      &id,
		Method:  method,
	}

	if params != nil {
		// Convert params to json.RawMessage if provided
		raw, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params: %w", err)
		}
		rawMsg := json.RawMessage(raw)
		msg.Params = &rawMsg
	}

	if c.logger != nil {
		c.logger.Logf("Client sending message: method=%s id=%v", method, id)
	}

	err := c.transport.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
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
