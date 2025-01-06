package client

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
)

type Client struct {
	transport transport.Transport
	nextID    uint64

	// Lifecycle management
	startOnce sync.Once
	closeOnce sync.Once
}

func NewClient(t transport.Transport) *Client {
	return &Client{transport: t}
}

// Start begins processing messages
func (c *Client) Start(ctx context.Context) error {
	var startErr error
	c.startOnce.Do(func() {
		startErr = c.transport.Start(ctx)
	})
	return startErr
}

// Close shuts down the client
func (c *Client) Close() error {
	var closeErr error
	c.closeOnce.Do(func() {
		closeErr = c.transport.Close()
	})
	return closeErr
}

// GetRouter returns the message router
func (c *Client) GetRouter() *transport.MessageRouter {
	return c.transport.GetRouter()
}

// Logf logs a formatted message
func (c *Client) Logf(format string, args ...interface{}) {
	c.transport.Logf(format, args...)
}

// SendRequest sends a request and waits for the response
func (c *Client) SendRequest(ctx context.Context, method string, params interface{}) (*types.Message, error) {
	// Generate request ID
	id := atomic.AddUint64(&c.nextID, 1)

	// Create request message
	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      &types.ID{Num: id},
		Method:  method,
	}

	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		raw := json.RawMessage(data)
		msg.Params = &raw
	}

	// Send the request
	if err := c.transport.Send(ctx, msg); err != nil {
		return nil, err
	}

	// Wait for response
	router := c.transport.GetRouter()
	for {
		select {
		case resp := <-router.Responses:
			if resp.ID != nil && resp.ID.Num == id {
				return resp, nil
			}
			// Not our response, put it back
			select {
			case router.Responses <- resp:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-router.Done():
			return nil, types.NewError(types.InternalError, "client closed")
		}
	}
}

// SendNotification sends a notification (no response expected)
func (c *Client) SendNotification(ctx context.Context, method string, params interface{}) error {
	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		Method:  method,
	}

	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return err
		}
		raw := json.RawMessage(data)
		msg.Params = &raw
	}

	return c.transport.Send(ctx, msg)
}
