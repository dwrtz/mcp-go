package sse

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/dwrtz/mcp-go/internal/testutil"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// getFreePort gets a free port from the OS
func getFreePort() (string, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "", err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return "", err
	}
	defer l.Close()
	return l.Addr().String(), nil
}

func TestSSETransport(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"TestBasicConnection", testBasicConnection},
		{"TestMessageExchange", testMessageExchange},
		{"TestReconnection", testReconnection},
		{"TestServerClose", testServerClose},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(t)
		})
	}
}

func testBasicConnection(t *testing.T) {
	addr, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := testutil.NewTestLogger(t)

	// Create server transport
	serverTransport := NewSSEServer(addr)
	serverTransport.SetLogger(logger)
	if err := serverTransport.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer serverTransport.Close()

	// Create client transport
	clientTransport := NewSSEClient(addr)
	clientTransport.SetLogger(logger)
	if err := clientTransport.Start(ctx); err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}
	defer clientTransport.Close()

	// Wait a bit to ensure connection is established
	time.Sleep(100 * time.Millisecond)

	// Check if transports are still running
	select {
	case <-serverTransport.Done():
		t.Error("Server transport closed unexpectedly")
	case <-clientTransport.Done():
		t.Error("Client transport closed unexpectedly")
	default:
		// OK - still running
	}
}

func testMessageExchange(t *testing.T) {
	addr, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := testutil.NewTestLogger(t)

	// Create server transport
	serverTransport := NewSSEServer(addr)
	serverTransport.SetLogger(logger)
	if err := serverTransport.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer serverTransport.Close()

	// Create client transport
	clientTransport := NewSSEClient(addr)
	clientTransport.SetLogger(logger)
	if err := clientTransport.Start(ctx); err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}
	defer clientTransport.Close()

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Channels to capture received messages
	serverRecvCh := make(chan *types.Message, 1)
	clientRecvCh := make(chan *types.Message, 1)

	// Listen on server router
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-serverTransport.GetRouter().Requests:
				if !ok {
					return
				}
				serverRecvCh <- msg
			case msg, ok := <-serverTransport.GetRouter().Notifications:
				if !ok {
					return
				}
				serverRecvCh <- msg
			}
		}
	}()

	// Listen on client router
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-clientTransport.GetRouter().Requests:
				if !ok {
					return
				}
				clientRecvCh <- msg
			case msg, ok := <-clientTransport.GetRouter().Notifications:
				if !ok {
					return
				}
				clientRecvCh <- msg
			}
		}
	}()

	// Create valid JSON-RPC messages (with "jsonrpc":"2.0")
	testMsg1 := testutil.CreateTestMessage(t, &types.ID{Num: 1}, "test", map[string]string{
		"from": "client",
	})
	testMsg2 := testutil.CreateTestMessage(t, &types.ID{Num: 2}, "test", map[string]string{
		"from": "server",
	})

	// Send messages both ways
	errCh := make(chan error, 2)
	go func() {
		errCh <- clientTransport.Send(ctx, testMsg1)
	}()
	go func() {
		errCh <- serverTransport.Send(ctx, testMsg2)
	}()

	// Wait for messages or timeout
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	received1 := false
	received2 := false

	for !received1 || !received2 {
		select {
		case err := <-errCh:
			if err != nil {
				t.Errorf("Failed to send message: %v", err)
			}
		case msg := <-serverRecvCh:
			testutil.AssertMessagesEqual(t, testMsg1, msg)
			received1 = true
		case msg := <-clientRecvCh:
			testutil.AssertMessagesEqual(t, testMsg2, msg)
			received2 = true
		case <-timer.C:
			t.Error("Timeout waiting for message exchange")
			return
		}
	}
}

func testReconnection(t *testing.T) {
	addr, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := testutil.NewTestLogger(t)

	// Create server transport
	serverTransport := NewSSEServer(addr)
	serverTransport.SetLogger(logger)
	if err := serverTransport.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer serverTransport.Close()

	// Create first client
	client := NewSSEClient(addr)
	client.SetLogger(logger)
	if err := client.Start(ctx); err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Close first client
	client.Close()
	time.Sleep(100 * time.Millisecond)

	// Create new client - should be accepted now that the first is gone
	client2 := NewSSEClient(addr)
	client2.SetLogger(logger)
	if err := client2.Start(ctx); err != nil {
		t.Fatalf("Failed to start second client: %v", err)
	}
	defer client2.Close()

	// Try to send message
	testMsg := testutil.CreateTestMessage(t, &types.ID{Num: 1}, "test", nil)
	if err := serverTransport.Send(ctx, testMsg); err != nil {
		t.Errorf("Failed to send after reconnection: %v", err)
	}
}

func testServerClose(t *testing.T) {
	addr, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := testutil.NewTestLogger(t)

	// Create server transport
	serverTransport := NewSSEServer(addr)
	serverTransport.SetLogger(logger)
	if err := serverTransport.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Create client transport
	clientTransport := NewSSEClient(addr)
	clientTransport.SetLogger(logger)
	if err := clientTransport.Start(ctx); err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}
	defer clientTransport.Close()

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Close server
	serverTransport.Close()

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Try to send message - should fail
	testMsg := testutil.CreateTestMessage(t, &types.ID{Num: 1}, "test", nil)
	if err := clientTransport.Send(ctx, testMsg); err == nil {
		t.Error("Expected error sending after server close, got none")
	}
}
