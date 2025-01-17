package client

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// NotificationHandler handles MCP notifications
type NotificationHandler func(ctx context.Context, params json.RawMessage)

type Client struct {
	transport transport.Transport
	nextID    uint64

	// Message handling
	notificationHandlers map[string]NotificationHandler
	handlerMu            sync.RWMutex // Protects notificationHandlers

	// Lifecycle management
	startOnce sync.Once
	closeOnce sync.Once
}

func NewClient(t transport.Transport) *Client {
	return &Client{
		transport:            t,
		notificationHandlers: make(map[string]NotificationHandler),
	}
}

// RegisterNotificationHandler registers a handler for a notification method
func (c *Client) RegisterNotificationHandler(method string, handler NotificationHandler) {
	c.handlerMu.Lock()
	defer c.handlerMu.Unlock()
	c.notificationHandlers[method] = handler
}

// Start begins processing messages
func (c *Client) Start(ctx context.Context) error {
	var startErr error
	c.startOnce.Do(func() {
		// Start transport
		if err := c.transport.Start(ctx); err != nil {
			startErr = err
			return
		}

		// Start message handling
		go c.handleMessages(ctx)
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

// handleMessages processes incoming messages from the transport
func (c *Client) handleMessages(ctx context.Context) {
	router := c.transport.GetRouter()
	for {
		select {
		case notif := <-router.Notifications:
			// Handle notification in a goroutine
			go c.handleNotification(ctx, notif)
		case <-ctx.Done():
			return
		case <-router.Done():
			return
		}
	}
}

func (c *Client) handleNotification(ctx context.Context, msg *types.Message) {
	if msg.Params == nil {
		c.Logf("Received notification without params: %s", msg.Method)
		return
	}

	c.handlerMu.RLock()
	handler, ok := c.notificationHandlers[msg.Method]
	c.handlerMu.RUnlock()

	if ok {
		handler(ctx, *msg.Params)
	} else {
		c.Logf("No handler registered for notification method: %s", msg.Method)
	}
}
