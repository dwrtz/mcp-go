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
type Options struct {
	// If these are nil, we'll default to os.Stdin/os.Stdout/os.Stderr
	Stdin  io.ReadCloser
	Stdout io.WriteCloser
	Stderr io.WriteCloser

	// Common transport options
	*transport.Options
}

// NewStdioTransport creates a new StdioTransport
func NewStdioTransport(opts *Options) *StdioTransport {
	if opts == nil {
		opts = &Options{
			Options: &transport.Options{},
		}
	}

	if opts.Options == nil {
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

	base := transport.NewBaseTransport()
	if opts.Options.Logger != nil {
		base.SetLogger(opts.Options.Logger)
	}

	t := &StdioTransport{
		BaseTransport: base,
		stdin:         opts.Stdin,
		stdout:        opts.Stdout,
		stderr:        opts.Stderr,
	}

	if opts.Options.Handler != nil {
		t.SetHandler(opts.Options.Handler)
	}

	return t
}

// Send implements Transport
func (t *StdioTransport) Send(ctx context.Context, msg *types.Message) error {
	t.mu.Lock()
	conn := t.Conn
	t.mu.Unlock()

	if conn == nil {
		return errors.New("transport not started")
	}

	if msg.ID != nil {
		t.BaseTransport.Logger.Logf("Sending request with ID %v: %s", msg.ID, msg.Method)
		var result interface{}
		callOpts := []jsonrpc2.CallOption{
			jsonrpc2.PickID(*msg.ID),
		}
		return conn.Call(ctx, msg.Method, msg.Params, &result, callOpts...)
	} else {
		t.BaseTransport.Logger.Logf("Sending notification: %s", msg.Method)
		return conn.Notify(ctx, msg.Method, msg.Params)
	}
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
