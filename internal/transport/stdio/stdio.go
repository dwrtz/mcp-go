package stdio

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
	"github.com/sourcegraph/jsonrpc2"
)

// StdioTransport implements transport.Transport using stdin/stdout
type StdioTransport struct {
	*transport.BaseTransport
	stdin  io.ReadCloser
	stdout io.WriteCloser
	stderr io.WriteCloser
	mu     sync.Mutex
}

// Options holds custom I/O streams for the StdioTransport
// plus any other relevant fields you want.
type Options struct {
	// If these are nil, we’ll default to os.Stdin/os.Stdout/os.Stderr.
	Stdin  io.ReadCloser
	Stdout io.WriteCloser
	Stderr io.WriteCloser

	*transport.Options
}

// NewStdioTransport creates a new StdioTransport using
// either custom I/O (if provided) or the default os.* streams.
func NewStdioTransport(opts *Options) *StdioTransport {
	// Defensive check so we never panic on opts == nil.
	if opts == nil {
		opts = &Options{
			Options: &transport.Options{},
		}
	} else if opts.Options == nil {
		// Ensure the nested Options is initialized
		opts.Options = &transport.Options{}
	}

	// Set default I/O if not provided
	if opts.Stdin == nil {
		opts.Stdin = os.Stdin
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	// Make the base transport
	base := transport.NewBaseTransport()

	t := &StdioTransport{
		BaseTransport: base,
		stdin:         opts.Stdin,
		stdout:        opts.Stdout,
		stderr:        opts.Stderr,
	}

	// If caller provided a Handler in transport.Options, set it
	if opts.Handler != nil {
		t.SetHandler(opts.Handler)
	}

	return t
}

// Start implements Transport
func (t *StdioTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	if t.IsClosed() {
		t.mu.Unlock()
		return errors.New("transport is closed")
	}

	// Create JSON-RPC stream
	stream := jsonrpc2.NewBufferedStream(stdioStream{t.stdin, t.stdout}, jsonrpc2.VSCodeObjectCodec{})

	// Create connection
	conn := jsonrpc2.NewConn(ctx, stream, jsonrpc2.HandlerWithError(t.Handle))
	t.Conn = conn
	t.mu.Unlock()

	// Wait for connection to close
	select {
	case <-conn.DisconnectNotify():
	case <-ctx.Done():
	}
	return nil
}

// Send implements Transport
func (t *StdioTransport) Send(ctx context.Context, msg *types.Message) error {
	t.mu.Lock()
	conn := t.Conn
	t.mu.Unlock()

	if conn == nil {
		return errors.New("transport not started")
	}

	// If an ID is present, treat it like a request; otherwise it's a notification.
	if msg.ID != nil {
		// "Blocking" JSON-RPC call, ignoring returned result:
		return conn.Call(ctx, msg.Method, msg.Params, nil /* no result needed */)
	} else {
		// One-way notification:
		return conn.Notify(ctx, msg.Method, msg.Params)
	}
}

// stdioStream implements jsonrpc2.Stream using stdin/stdout
type stdioStream struct {
	in  io.ReadCloser
	out io.WriteCloser
}

func (s stdioStream) Read(p []byte) (n int, err error) {
	return s.in.Read(p)
}

func (s stdioStream) Write(p []byte) (n int, err error) {
	n, err = s.out.Write(p)
	if err != nil {
		return n, err
	}
	if f, ok := s.out.(*os.File); ok {
		err = f.Sync()
	}
	return n, err
}

func (s stdioStream) Close() error {
	var err1, err2 error
	if s.in != nil {
		err1 = s.in.Close()
	}
	if s.out != nil {
		err2 = s.out.Close()
	}
	if err1 != nil {
		return err1
	}
	return err2
}
