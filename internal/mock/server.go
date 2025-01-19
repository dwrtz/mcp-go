package mock

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/internal/testutil"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// MockServer provides helper methods for testing MCP servers
type MockServer struct {
	T          *testing.T
	Transport  *MockTransport
	BaseServer *base.Server
	Context    context.Context
	CancelFunc context.CancelFunc
	Logger     *testutil.TestLogger
}

// NewMockServer creates a new MockServer with all dependencies initialized
func NewMockServer(t *testing.T) *MockServer {
	logger := testutil.NewTestLogger(t)
	mockTransport := NewMockTransport(logger)
	baseServer := base.NewServer(mockTransport)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	server := &MockServer{
		T:          t,
		Transport:  mockTransport,
		BaseServer: baseServer,
		Context:    ctx,
		CancelFunc: cancel,
		Logger:     logger,
	}

	// Start the transport and server
	if err := mockTransport.Start(ctx); err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}
	if err := baseServer.Start(ctx); err != nil {
		cancel()
		t.Fatalf("Failed to start server: %v", err)
	}

	return server
}

// Close cleans up resources
func (s *MockServer) Close() {
	s.CancelFunc()
	s.BaseServer.Close()
	s.Transport.Close()
}

// ExpectRequest waits for and handles a request with the given method
func (s *MockServer) ExpectRequest(method string, handler func(*types.Message) *types.Message) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)
		select {
		case msg := <-s.Transport.GetRouter().Requests:
			if msg.Method != method {
				s.T.Errorf("Expected method %s, got %s", method, msg.Method)
				return
			}

			if response := handler(msg); response != nil {
				s.Transport.SimulateReceive(s.Context, response)
			}

		case <-s.Context.Done():
			s.T.Error("Context cancelled while waiting for request")
		}
	}()

	return done
}

// CreateSuccessResponse creates a successful response message
func (s *MockServer) CreateSuccessResponse(msg *types.Message, result interface{}) *types.Message {
	response := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      msg.ID,
	}

	if result != nil {
		data, err := testutil.MarshalResult(result)
		if err != nil {
			s.T.Fatalf("Failed to marshal result: %v", err)
		}
		response.Result = data
	} else {
		emptyObj := json.RawMessage([]byte("{}"))
		response.Result = &emptyObj
	}

	return response
}

// CreateErrorResponse creates an error response message
func (s *MockServer) CreateErrorResponse(msg *types.Message, code int, message string, data interface{}) *types.Message {
	return &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      msg.ID,
		Error: &types.ErrorResponse{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// SimulateRequest simulates receiving a request from the client
func (s *MockServer) SimulateRequest(method string, params interface{}) (*types.Message, error) {
	var raw json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		raw = data
	} else {
		// Always use empty object if no params provided
		raw = []byte("{}")
	}

	request := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      &types.ID{Num: 1}, // Use a simple ID for testing
		Method:  method,
		Params:  &raw,
	}

	responseChan := make(chan *types.Message, 1)
	go func() {
		select {
		case resp := <-s.Transport.GetRouter().Responses:
			responseChan <- resp
		case <-time.After(time.Second):
			s.T.Error("Timeout waiting for response")
			close(responseChan)
		case <-s.Context.Done():
			s.T.Error("Context cancelled while waiting for response")
			close(responseChan)
		}
	}()

	s.Transport.SimulateReceive(s.Context, request)

	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(time.Second):
		return nil, ErrTimeoutWaitingForResponse
	case <-s.Context.Done():
		return nil, ErrContextCancelled
	}
}

// SimulateNotification simulates receiving a notification from the client
func (s *MockServer) SimulateNotification(method string, params interface{}) error {
	var raw json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return err
		}
		raw = data
	} else {
		// Always use empty object if no params provided
		raw = []byte("{}")
	}

	notification := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		Method:  method,
		Params:  &raw,
	}

	s.Transport.SimulateReceive(s.Context, notification)
	return nil
}

// WaitForCallback waits for a callback to be invoked with timeout
func (s *MockServer) WaitForCallback(callback func(done chan<- struct{})) error {
	done := make(chan struct{})

	go callback(done)

	select {
	case <-done:
		return nil
	case <-time.After(time.Second):
		return ErrTimeoutWaitingForCallback
	case <-s.Context.Done():
		return ErrContextCancelled
	}
}

// AssertNoErrors checks the error channel for any errors
func (s *MockServer) AssertNoErrors(t *testing.T) {
	select {
	case err := <-s.Transport.GetRouter().Errors:
		t.Errorf("Unexpected error: %v", err)
	default:
		// No errors - success
	}
}

// Common test errors
var (
	ErrTimeoutWaitingForResponse = errors.New("timeout waiting for response")
)
