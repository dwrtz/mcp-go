package server

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/dwrtz/mcp-go/internal/message"
	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
)

type Server struct {
	*message.MessageRouter
	transport transport.Transport

	// Lifecycle management
	startOnce sync.Once
	closeOnce sync.Once
}

func NewServer(t transport.Transport, logger transport.Logger) *Server {
	s := &Server{
		MessageRouter: message.NewMessageRouter(logger),
		transport:     t,
	}
	t.SetHandler(s)
	return s
}

// Start begins processing messages
func (s *Server) Start(ctx context.Context) error {
	var startErr error
	s.startOnce.Do(func() {
		startErr = s.transport.Start(ctx)
	})
	return startErr
}

// Close shuts down the server
func (s *Server) Close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		s.MessageRouter.Close()
		closeErr = s.transport.Close()
	})
	return closeErr
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
