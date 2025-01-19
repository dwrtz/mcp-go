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

// MockClient provides helper methods for testing MCP clients
type MockClient struct {
	T          *testing.T
	Transport  *MockTransport
	BaseClient *base.Client
	Context    context.Context
	CancelFunc context.CancelFunc
	Logger     *testutil.TestLogger
}

// NewMockClient creates a new MockClient with all dependencies initialized
func NewMockClient(t *testing.T) *MockClient {
	logger := testutil.NewTestLogger(t)
	mockTransport := NewMockTransport(logger)
	baseClient := base.NewClient(mockTransport)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	client := &MockClient{
		T:          t,
		Transport:  mockTransport,
		BaseClient: baseClient,
		Context:    ctx,
		CancelFunc: cancel,
		Logger:     logger,
	}

	// Start the transport and client
	if err := mockTransport.Start(ctx); err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}
	if err := baseClient.Start(ctx); err != nil {
		cancel()
		t.Fatalf("Failed to start client: %v", err)
	}

	return client
}

// Close cleans up resources
func (c *MockClient) Close() {
	c.CancelFunc()
	c.BaseClient.Close()
	c.Transport.Close()
}

// ExpectRequest waits for and handles a request with the given method
func (c *MockClient) ExpectRequest(method string, handler func(*types.Message) *types.Message) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)
		select {
		case msg := <-c.Transport.GetRouter().Requests:
			if msg.Method != method {
				c.T.Errorf("Expected method %s, got %s", method, msg.Method)
				return
			}

			if response := handler(msg); response != nil {
				c.Transport.SimulateReceive(c.Context, response)
			}

		case <-c.Context.Done():
			c.T.Error("Context cancelled while waiting for request")
		}
	}()

	return done
}

// CreateSuccessResponse creates a successful response message
func (c *MockClient) CreateSuccessResponse(msg *types.Message, result interface{}) *types.Message {
	response := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      msg.ID,
	}

	if result != nil {
		data, err := testutil.MarshalResult(result)
		if err != nil {
			c.T.Fatalf("Failed to marshal result: %v", err)
		}
		response.Result = data
	} else {
		emptyObj := json.RawMessage([]byte("{}"))
		response.Result = &emptyObj
	}

	return response
}

// CreateErrorResponse creates an error response message
func (c *MockClient) CreateErrorResponse(msg *types.Message, code int, message string, data interface{}) *types.Message {
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

// SimulateNotification simulates receiving a notification from the server
func (c *MockClient) SimulateNotification(method string, params interface{}) error {
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

	c.Transport.SimulateReceive(c.Context, notification)
	return nil
}

// WaitForCallback waits for a callback to be invoked with timeout
func (c *MockClient) WaitForCallback(callback func(done chan<- struct{})) error {
	done := make(chan struct{})

	go callback(done)

	select {
	case <-done:
		return nil
	case <-time.After(time.Second):
		return ErrTimeoutWaitingForCallback
	case <-c.Context.Done():
		return ErrContextCancelled
	}
}

// AssertNoErrors checks the error channel for any errors
func (c *MockClient) AssertNoErrors(t *testing.T) {
	select {
	case err := <-c.Transport.GetRouter().Errors:
		t.Errorf("Unexpected error: %v", err)
	default:
		// No errors - success
	}
}

// Common test errors
var (
	ErrTimeoutWaitingForCallback = errors.New("timeout waiting for callback")
	ErrContextCancelled          = errors.New("context cancelled while waiting for callback")
)
