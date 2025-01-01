package stdio

import (
	"context"
	"io"
	"sync"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
	"github.com/sourcegraph/jsonrpc2"
)

type StdioTransport struct {
	handler transport.MessageHandler
	conn    *jsonrpc2.Conn
	done    chan struct{}
	mu      sync.Mutex
	logger  transport.Logger
}

func NewStdioTransport(stdin io.ReadCloser, stdout io.WriteCloser, logger transport.Logger) *StdioTransport {
	return &StdioTransport{
		done:   make(chan struct{}),
		logger: logger,
	}
}

func (t *StdioTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Create JSON-RPC stream
	stream := jsonrpc2.NewBufferedStream(stdioStream{t.stdin, t.stdout}, jsonrpc2.VSCodeObjectCodec{})

	// Create connection and set our jsonrpc2.Handler implementation
	t.conn = jsonrpc2.NewConn(ctx, stream, &jsonRPCHandler{t})

	// Wait for shutdown
	go func() {
		select {
		case <-t.conn.DisconnectNotify():
			t.Close()
		case <-ctx.Done():
			t.Close()
		}
	}()

	return nil
}

func (t *StdioTransport) Send(ctx context.Context, msg *types.Message) error {
	t.mu.Lock()
	conn := t.conn
	t.mu.Unlock()

	if conn == nil {
		return types.NewError(types.InternalError, "transport not started")
	}

	if msg.Method != "" {
		// This is a request or notification
		if msg.ID != nil {
			// Request
			var result interface{}
			return conn.Call(ctx, msg.Method, msg.Params, &result, jsonrpc2.PickID(*msg.ID))
		}
		// Notification
		return conn.Notify(ctx, msg.Method, msg.Params)
	}

	// This is a response
	if msg.Error != nil {
		return conn.ReplyWithError(ctx, *msg.ID, &jsonrpc2.Error{
			Code:    msg.Error.Code,
			Message: msg.Error.Message,
			Data:    msg.Error.Data,
		})
	}
	return conn.Reply(ctx, *msg.ID, msg.Result)
}

func (t *StdioTransport) SetHandler(handler transport.MessageHandler) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handler = handler
}

func (t *StdioTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	select {
	case <-t.done:
		return nil
	default:
		close(t.done)
		if t.conn != nil {
			return t.conn.Close()
		}
		return nil
	}
}

func (t *StdioTransport) Done() <-chan struct{} {
	return t.done
}

// jsonRPCHandler implements jsonrpc2.Handler
type jsonRPCHandler struct {
	transport *StdioTransport
}

func (h *jsonRPCHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	t := h.transport

	// Convert JSON-RPC request to MCP message
	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		Method:  req.Method,
		Params:  req.Params,
	}
	if !req.ID.IsNotify() {
		msg.ID = &req.ID
	}

	// Get handler (with mutex protection)
	t.mu.Lock()
	handler := t.handler
	t.mu.Unlock()

	if handler == nil {
		if !req.ID.IsNotify() {
			err := conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
				Code:    types.MethodNotFound,
				Message: "no handler registered",
			})
			if err != nil {
				t.logger.Logf("Failed to send error response: %v", err)
			}
		}
		return
	}

	// Handle the message
	resp, err := handler.Handle(ctx, msg)
	if err != nil {
		if !req.ID.IsNotify() {
			// Convert to JSON-RPC error if needed
			var rpcErr *jsonrpc2.Error
			if e, ok := err.(*jsonrpc2.Error); ok {
				rpcErr = e
			} else {
				rpcErr = &jsonrpc2.Error{
					Code:    types.InternalError,
					Message: err.Error(),
				}
			}

			if err := conn.ReplyWithError(ctx, req.ID, rpcErr); err != nil {
				t.logger.Logf("Failed to send error response: %v", err)
			}
		}
		return
	}

	// Send response if this was a request (not a notification) and there is a response
	if resp != nil && !req.ID.IsNotify() {
		if err := conn.Reply(ctx, req.ID, resp.Result); err != nil {
			t.logger.Logf("Failed to send response: %v", err)
		}
	}
}
