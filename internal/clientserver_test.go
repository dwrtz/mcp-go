// internal/clientserver_test.go

package test

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/dwrtz/mcp-go/internal/client"
	"github.com/dwrtz/mcp-go/internal/server"
	"github.com/dwrtz/mcp-go/internal/testutil"
	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/internal/transport/stdio"
)

// nopWriteCloser wraps an io.Writer and provides a no-op Close method
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

func TestPingPong(t *testing.T) {
	logger := testutil.NewTestLogger(t)

	// Create pipes to simulate in-proc stdio for server and client
	serverStdinR, serverStdinW := io.Pipe()
	serverStdoutR, serverStdoutW := io.Pipe()
	clientStdinR, clientStdinW := io.Pipe()
	clientStdoutR, clientStdoutW := io.Pipe()

	// Wire them up so server's stdin is client's stdout, and vice versa
	go func() {
		defer serverStdinW.Close()
		_, err := io.Copy(serverStdinW, clientStdoutR)
		if err != nil {
			logger.Logf("Server stdin copy error: %v", err)
		}
	}()
	go func() {
		defer clientStdinW.Close()
		_, err := io.Copy(clientStdinW, serverStdoutR)
		if err != nil {
			logger.Logf("Client stdin copy error: %v", err)
		}
	}()

	// Create devNull as a WriteCloser for stderr
	devNull := nopWriteCloser{io.Discard}

	// Context with timeout for the entire test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create and start server
	serverTransport := stdio.NewStdioTransport(&stdio.Options{
		Stdin:  serverStdinR,
		Stdout: serverStdoutW,
		Stderr: devNull,
		Options: &transport.Options{
			Logger: logger,
		},
	})
	srv := server.NewServer(serverTransport)
	serverTransport.SetHandler(&server.Handler{Logger: logger})

	// Create and start client
	clientTransport := stdio.NewStdioTransport(&stdio.Options{
		Stdin:  clientStdinR,
		Stdout: clientStdoutW,
		Stderr: devNull,
		Options: &transport.Options{
			Logger: logger,
		},
	})
	cli := client.NewClient(clientTransport)

	// Start server first
	logger.Logf("Starting server...")
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("server.Start() error: %v", err)
	}

	// Use WaitGroup to track completion
	var wg sync.WaitGroup
	wg.Add(1)

	// Start client with handler
	logger.Logf("Starting client...")
	pingID := "1" // Changed to match our numeric ID
	if err := cli.Start(ctx, pingID); err != nil {
		t.Fatalf("client.Start() error: %v", err)
	}

	// Listen for the client to close once "ping" is done
	go func() {
		defer wg.Done()
		<-cli.Done()
		logger.Logf("Client done channel closed")
	}()

	// Give transports a moment to initialize
	time.Sleep(200 * time.Millisecond)

	// Send ping
	logger.Logf("Sending ping...")
	if err := cli.Ping(ctx, pingID); err != nil {
		t.Fatalf("failed to send ping: %v", err)
	}

	// Wait for completion or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Logf("Test completed successfully")
		// Clean up
		if err := cli.Close(); err != nil {
			logger.Logf("Warning: client.Close() error: %v", err)
		}
		if err := srv.Close(); err != nil {
			logger.Logf("Warning: server.Close() error: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("test timed out: %v\nLog output:\n%s", ctx.Err(), logger.String())
	}
}
