package base

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/dwrtz/mcp-go/internal/testutil"
	"github.com/dwrtz/mcp-go/internal/transport/stdio"
	"github.com/dwrtz/mcp-go/pkg/methods"
)

func TestPingPong(t *testing.T) {
	logger := testutil.NewTestLogger(t)

	// Create pipes to simulate in-proc stdio
	serverStdinR, serverStdinW := io.Pipe()
	serverStdoutR, serverStdoutW := io.Pipe()
	clientStdinR, clientStdinW := io.Pipe()
	clientStdoutR, clientStdoutW := io.Pipe()

	// Wire up pipes
	go func() {
		defer serverStdinW.Close()
		io.Copy(serverStdinW, clientStdoutR)
	}()
	go func() {
		defer clientStdinW.Close()
		io.Copy(clientStdinW, serverStdoutR)
	}()

	// Create transports
	serverTransport := stdio.NewStdioTransport(serverStdinR, serverStdoutW, logger)
	clientTransport := stdio.NewStdioTransport(clientStdinR, clientStdoutW, logger)

	// Create server and client
	srv := NewBase(serverTransport)
	cli := NewBase(clientTransport)

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Register ping handler on server
	srv.RegisterRequestHandler(methods.Ping, func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		logger.Logf("Server received ping, sending response")
		return map[string]string{"status": "ok"}, nil
	})

	// Start server and client
	logger.Logf("Starting server...")
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("server.Start() error: %v", err)
	}

	logger.Logf("Starting client...")
	if err := cli.Start(ctx); err != nil {
		t.Fatalf("client.Start() error: %v", err)
	}

	// Send ping request
	logger.Logf("Sending ping request...")
	resp, err := cli.SendRequest(ctx, methods.Ping, nil)
	if err != nil {
		t.Fatalf("SendRequest error: %v", err)
	}

	// Verify response
	var result map[string]string
	if err := json.Unmarshal(*resp.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", result["status"])
	}
	logger.Logf("Client received expected response")

	// Clean up
	cli.Close()
	srv.Close()
}

func TestNotifications(t *testing.T) {
	logger := testutil.NewTestLogger(t)

	// Create pipes to simulate in-proc stdio
	serverStdinR, serverStdinW := io.Pipe()
	serverStdoutR, serverStdoutW := io.Pipe()
	clientStdinR, clientStdinW := io.Pipe()
	clientStdoutR, clientStdoutW := io.Pipe()

	// Wire up pipes
	go func() {
		defer serverStdinW.Close()
		io.Copy(serverStdinW, clientStdoutR)
	}()
	go func() {
		defer clientStdinW.Close()
		io.Copy(clientStdinW, serverStdoutR)
	}()

	// Create transports
	serverTransport := stdio.NewStdioTransport(serverStdinR, serverStdoutW, logger)
	clientTransport := stdio.NewStdioTransport(clientStdinR, clientStdoutW, logger)

	// Create server and client
	srv := NewBase(serverTransport)
	cli := NewBase(clientTransport)

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Channel to track received notifications
	receivedNotification := make(chan struct{})

	// Register notification handler on client
	cli.RegisterNotificationHandler("test/notification", func(ctx context.Context, params json.RawMessage) {
		var msg string
		if err := json.Unmarshal(params, &msg); err != nil {
			t.Errorf("Failed to unmarshal notification params: %v", err)
			return
		}
		if msg != "hello" {
			t.Errorf("Expected message 'hello', got '%s'", msg)
		}
		close(receivedNotification)
	})

	// Start server and client
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("server.Start() error: %v", err)
	}
	if err := cli.Start(ctx); err != nil {
		t.Fatalf("client.Start() error: %v", err)
	}

	// Send notification from server to client
	logger.Logf("Server sending notification...")
	if err := srv.SendNotification(ctx, "test/notification", "hello"); err != nil {
		t.Fatalf("SendNotification error: %v", err)
	}

	// Wait for notification to be received
	select {
	case <-receivedNotification:
		logger.Logf("Client received notification")
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for notification")
	}

	// Clean up
	cli.Close()
	srv.Close()
}
