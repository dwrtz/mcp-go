package base

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// RequestHandler handles MCP requests and returns a response
type RequestHandler func(ctx context.Context, params json.RawMessage) (interface{}, error)

// NotificationHandler handles MCP notifications
type NotificationHandler func(ctx context.Context, params json.RawMessage)

// Base is a base abstraction for MCP clients and servers
type Base struct {
	transport transport.Transport
	nextID    uint64

	// Message handling
	requestHandlers      map[string]RequestHandler
	notificationHandlers map[string]NotificationHandler
	handlerMu            sync.RWMutex // Protects notificationHandlers

	// Lifecycle management
	startOnce sync.Once
	closeOnce sync.Once
}

// NewBase creates a new base instance
func NewBase(t transport.Transport) *Base {
	return &Base{
		transport:            t,
		requestHandlers:      make(map[string]RequestHandler),
		notificationHandlers: make(map[string]NotificationHandler),
	}
}

// RegisterRequestHandler registers a handler for a request method
func (b *Base) RegisterRequestHandler(method string, handler RequestHandler) {
	b.handlerMu.Lock()
	defer b.handlerMu.Unlock()
	b.requestHandlers[method] = handler
}

// RegisterNotificationHandler registers a handler for a notification method
func (b *Base) RegisterNotificationHandler(method string, handler NotificationHandler) {
	b.handlerMu.Lock()
	defer b.handlerMu.Unlock()
	b.notificationHandlers[method] = handler
}

// Start begins processing messages
func (b *Base) Start(ctx context.Context) error {
	var startErr error
	b.startOnce.Do(func() {
		// Start transport
		if err := b.transport.Start(ctx); err != nil {
			startErr = err
			return
		}

		// Start message handling
		go b.handleMessages(ctx)
	})
	return startErr
}

// Close shuts down the client
func (b *Base) Close() error {
	var closeErr error
	b.closeOnce.Do(func() {
		closeErr = b.transport.Close()
	})
	return closeErr
}

// GetRouter returns the message router
func (b *Base) GetRouter() *transport.MessageRouter {
	return b.transport.GetRouter()
}

// Logf logs a formatted message
func (b *Base) Logf(format string, args ...interface{}) {
	b.transport.Logf(format, args...)
}

// SendRequest sends a request and waits for the response
func (b *Base) SendRequest(ctx context.Context, method string, params interface{}) (*types.Message, error) {
	// Generate request ID
	id := atomic.AddUint64(&b.nextID, 1)

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
	if err := b.transport.Send(ctx, msg); err != nil {
		return nil, err
	}

	// Wait for response
	router := b.transport.GetRouter()
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

// SendResponse sends a response to a request
func (b *Base) SendResponse(ctx context.Context, reqID types.ID, result interface{}, err error) error {
	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      &reqID,
	}

	if err != nil {
		if mcpErr, ok := err.(*types.ErrorResponse); ok {
			msg.Error = mcpErr
		} else {
			msg.Error = types.NewError(types.InternalError, err.Error())
		}
	} else if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return err
		}
		raw := json.RawMessage(data)
		msg.Result = &raw
	}

	return b.transport.Send(ctx, msg)
}

// SendNotification sends a notification (no response expected)
func (b *Base) SendNotification(ctx context.Context, method string, params interface{}) error {
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

	return b.transport.Send(ctx, msg)
}

// handleMessages processes incoming messages from the transport
func (b *Base) handleMessages(ctx context.Context) {
	router := b.transport.GetRouter()
	for {
		select {
		case req, ok := <-router.Requests:
			if !ok {
				return
			}
			// Handle request in a goroutine
			go b.handleRequest(ctx, req)
		case notif, ok := <-router.Notifications:
			if !ok {
				return
			}
			// Handle notification in a goroutine
			go b.handleNotification(ctx, notif)
		case <-ctx.Done():
			return
		case <-router.Done():
			return
		}
	}
}

// handleRequest handles incoming requests
func (b *Base) handleRequest(ctx context.Context, msg *types.Message) {
	if msg.ID == nil {
		b.Logf("Received request without ID: %s", msg.Method)
		return
	}

	if msg.Params == nil {
		respErr := types.NewError(types.InvalidParams,
			fmt.Sprintf("missing params for method=%q, requestID=%v", msg.Method, *msg.ID))
		_ = b.SendResponse(ctx, *msg.ID, nil, respErr)
		return
	}

	b.handlerMu.RLock()
	handler, ok := b.requestHandlers[msg.Method]
	b.handlerMu.RUnlock()

	if ok {
		result, err := handler(ctx, *msg.Params)
		_ = b.SendResponse(ctx, *msg.ID, result, err)
		return
	}

	// Method not found
	respErr := types.NewError(types.MethodNotFound,
		fmt.Sprintf("method not found: %q (requestID=%v)", msg.Method, *msg.ID))
	_ = b.SendResponse(ctx, *msg.ID, nil, respErr)
}

// handleNotification handles incoming notifications
func (b *Base) handleNotification(ctx context.Context, msg *types.Message) {
	if msg.Params == nil {
		b.Logf("Received notification without params: %s", msg.Method)
		return
	}

	b.handlerMu.RLock()
	handler, ok := b.notificationHandlers[msg.Method]
	b.handlerMu.RUnlock()

	if ok {
		handler(ctx, *msg.Params)
	} else {
		b.Logf("No handler registered for notification method: %s", msg.Method)
	}
}
