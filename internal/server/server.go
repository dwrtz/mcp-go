package server

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/dwrtz/mcp-go/internal/transport"
)

// Server is a minimal wrapper that starts/stops the transport, which
// already manages JSON-RPC connections. The transport has a Handler
// (see handler.go), but this struct organizes lifecycle.
type Server struct {
	transport transport.Transport
	done      chan struct{}
	mu        sync.Mutex
	closed    bool
}

// NewServer creates a new Server with the given Transport. The transport
// should already have a Handler set (e.g. &Handler{} from handler.go).
func NewServer(t transport.Transport) *Server {
	return &Server{
		transport: t,
		done:      make(chan struct{}),
	}
}

// Start runs transport.Start in a goroutine and waits until done.
func (s *Server) Start(ctx context.Context) error {
	go func() {
		err := s.transport.Start(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "server transport stopped: %v\n", err)
		}
	}()
	return nil
}

// Close signals the transport to shut down and cleans up resources.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	close(s.done)
	return s.transport.Close()
}
