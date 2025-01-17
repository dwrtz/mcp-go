package server

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// RequestHandler handles MCP requests and returns a response
type RequestHandler func(ctx context.Context, params json.RawMessage) (interface{}, error)

// NotificationHandler handles MCP notifications
type NotificationHandler func(ctx context.Context, params json.RawMessage)

type Server struct {
	transport transport.Transport

	// Message handling
	requestHandlers      map[string]RequestHandler
	notificationHandlers map[string]NotificationHandler
	handlerMu            sync.RWMutex // Protects both handler maps

	// Lifecycle management
	startOnce sync.Once
	closeOnce sync.Once
}

func NewServer(t transport.Transport) *Server {
	return &Server{
		transport:            t,
		requestHandlers:      make(map[string]RequestHandler),
		notificationHandlers: make(map[string]NotificationHandler),
	}
}

// RegisterRequestHandler registers a handler for a request method
func (s *Server) RegisterRequestHandler(method string, handler RequestHandler) {
	s.handlerMu.Lock()
	defer s.handlerMu.Unlock()
	s.requestHandlers[method] = handler
}

// RegisterNotificationHandler registers a handler for a notification method
func (s *Server) RegisterNotificationHandler(method string, handler NotificationHandler) {
	s.handlerMu.Lock()
	defer s.handlerMu.Unlock()
	s.notificationHandlers[method] = handler
}

// Start begins processing messages
func (s *Server) Start(ctx context.Context) error {
	var startErr error
	s.startOnce.Do(func() {
		// Start transport
		if err := s.transport.Start(ctx); err != nil {
			startErr = err
			return
		}

		// Start message handling
		go s.handleMessages(ctx)
	})
	return startErr
}

// Close shuts down the server
func (s *Server) Close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		router := s.transport.GetRouter()
		router.Close()
		closeErr = s.transport.Close()
	})
	return closeErr
}

// GetRouter returns the message router
func (s *Server) GetRouter() *transport.MessageRouter {
	return s.transport.GetRouter()
}

// Logf logs a formatted message
func (s *Server) Logf(format string, args ...interface{}) {
	s.transport.Logf(format, args...)
}

// SendResponse sends a response to a request
func (s *Server) SendResponse(ctx context.Context, reqID types.ID, result interface{}, err error) error {
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

	return s.transport.Send(ctx, msg)
}

// SendNotification sends a notification to the client
func (s *Server) SendNotification(ctx context.Context, method string, params interface{}) error {
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

	return s.transport.Send(ctx, msg)
}

// handleMessages processes incoming messages from the transport
func (s *Server) handleMessages(ctx context.Context) {
	router := s.transport.GetRouter()
	for {
		select {
		case req := <-router.Requests:
			// Handle request in a goroutine
			go s.handleRequest(ctx, req)
		case notif := <-router.Notifications:
			// Handle notification in a goroutine
			go s.handleNotification(ctx, notif)
		case <-ctx.Done():
			return
		case <-router.Done():
			return
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, msg *types.Message) {
	if msg.ID == nil {
		s.Logf("Received request without ID: %s", msg.Method)
		return
	}

	if msg.Params == nil {
		err := types.NewError(types.InvalidParams, "missing params")
		s.SendResponse(ctx, *msg.ID, nil, err)
		return
	}

	s.handlerMu.RLock()
	handler, ok := s.requestHandlers[msg.Method]
	s.handlerMu.RUnlock()

	if ok {
		result, err := handler(ctx, *msg.Params)
		s.SendResponse(ctx, *msg.ID, result, err)
		return
	}

	// Method not found
	err := types.NewError(types.MethodNotFound, "method not found: "+msg.Method)
	s.SendResponse(ctx, *msg.ID, nil, err)
}

func (s *Server) handleNotification(ctx context.Context, msg *types.Message) {
	if msg.Params == nil {
		s.Logf("Received notification without params: %s", msg.Method)
		return
	}

	s.handlerMu.RLock()
	handler, ok := s.notificationHandlers[msg.Method]
	s.handlerMu.RUnlock()

	if ok {
		handler(ctx, *msg.Params)
	} else {
		s.Logf("No handler registered for notification method: %s", msg.Method)
	}
}
