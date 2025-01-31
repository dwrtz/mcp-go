package stdio

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/logger"
	"github.com/dwrtz/mcp-go/pkg/types"
	"github.com/sourcegraph/jsonrpc2"
)

// stdioStream implements jsonrpc2.Stream using an io.Reader + io.Writer
type stdioStream struct {
	in  io.ReadCloser
	out io.WriteCloser
}

func (s stdioStream) Read(p []byte) (int, error) {
	return s.in.Read(p)
}

func (s stdioStream) Write(p []byte) (int, error) {
	return s.out.Write(p)
}

func (s stdioStream) Close() error {
	errIn := s.in.Close()
	errOut := s.out.Close()
	if errIn != nil {
		return errIn
	}
	return errOut
}

// StdioTransport is a Transport implementation that reads from an io.ReadCloser
// and writes to an io.WriteCloser using the jsonrpc2 library.
type StdioTransport struct {
	router *transport.MessageRouter
	conn   *jsonrpc2.Conn
	done   chan struct{}
	mu     sync.Mutex
	logger *logger.Logger

	stdin  io.ReadCloser
	stdout io.WriteCloser
}

// NewStdioTransport constructs a transport from a read/write pair (usually pipes).
func NewStdioTransport(stdin io.ReadCloser, stdout io.WriteCloser) *StdioTransport {
	return &StdioTransport{
		router: transport.NewMessageRouter(),
		done:   make(chan struct{}),
		logger: nil,
		stdin:  stdin,
		stdout: stdout,
	}
}

// Start kicks off the jsonrpc2 listener in a background goroutine.
func (t *StdioTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Create JSON-RPC stream over stdin/stdout
	stream := jsonrpc2.NewPlainObjectStream(stdioStream{in: t.stdin, out: t.stdout})

	// Create the JSON-RPC handler
	handler := jsonRPCHandler{transport: t}

	// Create the connection. The custom handler below routes messages to t.router.
	t.conn = jsonrpc2.NewConn(ctx, stream, &handler)

	// Watch for disconnect
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

// Send sends a single JSON-RPC message. If it’s a request, we wait for a response.
func (t *StdioTransport) Send(ctx context.Context, msg *types.Message) error {
	t.Logf("Sending message: %+v", msg)

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn == nil {
		return types.NewError(types.InternalError, "transport not started")
	}

	// If msg.Method is non-empty, this is either a request or notification:
	if msg.Method != "" {
		// If msg.ID is set, it's a request
		if msg.ID != nil {
			var rawResult json.RawMessage
			err := t.conn.Call(ctx, msg.Method, msg.Params, &rawResult)
			if err != nil {
				// Convert jsonrpc2.Error → types.ErrorResponse if needed
				if rpcErr, ok := err.(*jsonrpc2.Error); ok {
					return types.NewError(int(rpcErr.Code), rpcErr.Message, rpcErr.Data)
				}
				return err
			}

			// Construct a synthetic "response" Message for the router
			response := &types.Message{
				JSONRPC: types.JSONRPCVersion,
				ID:      msg.ID,
				Result:  &rawResult,
			}
			t.router.Handle(ctx, response)
			return nil
		}
		// Otherwise it's a notification
		return t.conn.Notify(ctx, msg.Method, msg.Params)
	}

	// If msg.Method = "", then it's a response
	if msg.Error != nil {
		// Convert to jsonrpc2.Error
		var rawData *json.RawMessage
		if msg.Error.Data != nil {
			data, err := json.Marshal(msg.Error.Data)
			if err != nil {
				return err
			}
			raw := json.RawMessage(data)
			rawData = &raw
		}
		return t.conn.ReplyWithError(ctx, *msg.ID, &jsonrpc2.Error{
			Code:    int64(msg.Error.Code),
			Message: msg.Error.Message,
			Data:    rawData,
		})
	}

	return t.conn.Reply(ctx, *msg.ID, msg.Result)
}

// GetRouter returns the transport's MessageRouter
func (t *StdioTransport) GetRouter() *transport.MessageRouter {
	return t.router
}

// Close shuts down the connection + signals done
func (t *StdioTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	select {
	case <-t.done:
		// already closed
		return nil
	default:
		close(t.done)
		if t.conn != nil {
			return t.conn.Close()
		}
		return nil
	}
}

// Done returns a channel closed when transport is closed
func (t *StdioTransport) Done() <-chan struct{} {
	return t.done
}

// Logf logs a formatted string if a logger is set
func (t *StdioTransport) Logf(format string, args ...interface{}) {
	if t.logger != nil {
		(*t.logger).Logf(format, args...)
	}
}

// SetLogger sets a logger for debug printing
func (t *StdioTransport) SetLogger(l logger.Logger) {
	t.logger = &l
	t.router.SetLogger(l)
}

// -----------------------------------------------------------------------------
// jsonRPCHandler is the “glue” that takes JSON-RPC 2.0 messages from the
// sourcegraph/jsonrpc2 library and translates them into our internal
// "MCP types.Message" struct, then passes them into t.router.
// -----------------------------------------------------------------------------
type jsonRPCHandler struct {
	transport *StdioTransport
}

func (h *jsonRPCHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	h.transport.Logf("Received message: %+v", req)

	// Convert the request to an MCP-style message
	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		Method:  req.Method,
		Params:  req.Params,
	}
	if !req.Notif {
		// If it's not a notification, it has an ID
		msg.ID = &req.ID
	}

	// Route the message to the channels in the router (Requests, Notifications, or Responses).
	h.transport.router.Handle(ctx, msg)
}
