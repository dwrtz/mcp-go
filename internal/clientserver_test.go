package test

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/dwrtz/mcp-go/internal/client"
	"github.com/dwrtz/mcp-go/internal/server"
	"github.com/dwrtz/mcp-go/internal/testutil"
	"github.com/dwrtz/mcp-go/internal/transport/stdio"
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
	srv := server.NewServer(serverTransport, logger)
	cli := client.NewClient(clientTransport, logger)

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start server handler
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case req := <-srv.Requests:
				if req.Method == "ping" {
					logger.Logf("Server received ping, sending response")
					err := srv.SendResponse(ctx, *req.ID, map[string]string{"status": "ok"}, nil)
					if err != nil {
						logger.Logf("Error sending response: %v", err)
					}
				}
			case <-srv.Done():
				return
			case <-ctx.Done():
				return
			}
		}
	}()

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
	resp, err := cli.SendRequest(ctx, "ping", nil)
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
	wg.Wait()
}
