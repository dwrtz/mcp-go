package stdio

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
	"github.com/sourcegraph/jsonrpc2"
)

// stdioStream implements jsonrpc2.Stream using stdin/stdout
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

type StdioTransport struct {
	router *transport.MessageRouter
	conn   *jsonrpc2.Conn
	done   chan struct{}
	mu     sync.Mutex
	logger transport.Logger
	stdin  io.ReadCloser
	stdout io.WriteCloser
}

func NewStdioTransport(stdin io.ReadCloser, stdout io.WriteCloser, logger transport.Logger) *StdioTransport {
	return &StdioTransport{
		router: transport.NewMessageRouter(logger),
		done:   make(chan struct{}),
		logger: logger,
		stdin:  stdin,
		stdout: stdout,
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
	defer t.mu.Unlock()

	if t.conn == nil {
		return types.NewError(types.InternalError, "transport not started")
	}

	if msg.Method != "" {
		// This is a request or notification
		if msg.ID != nil {
			// This is a request - we need to handle the response
			var rawResult json.RawMessage
			err := t.conn.Call(ctx, msg.Method, msg.Params, &rawResult)
			if err != nil {
				// Convert jsonrpc2.Error to types.ErrorResponse if needed
				if rpcErr, ok := err.(*jsonrpc2.Error); ok {
					return types.NewError(int(rpcErr.Code), rpcErr.Message, rpcErr.Data)
				}
				return err
			}

			// Create response message
			response := &types.Message{
				JSONRPC: types.JSONRPCVersion,
				ID:      msg.ID,
				Result:  &rawResult,
			}

			// Route the response through the handler
			t.router.Handle(ctx, response)

			return nil
		}
		// This is a notification
		return t.conn.Notify(ctx, msg.Method, msg.Params)
	}

	// This is a response
	if msg.Error != nil {
		// Convert error data to RawMessage if present
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

func (t *StdioTransport) GetRouter() *transport.MessageRouter {
	return t.router
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

func (t *StdioTransport) Logf(format string, args ...interface{}) {
	t.logger.Logf(format, args...)
}

// jsonRPCHandler implements jsonrpc2.Handler
type jsonRPCHandler struct {
	transport *StdioTransport
}

func (h *jsonRPCHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	// Convert JSON-RPC request to MCP message
	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		Method:  req.Method,
		Params:  req.Params,
	}
	if !req.Notif {
		msg.ID = &req.ID
	}

	// Route the message to handler channels
	h.transport.router.Handle(ctx, msg)
}
