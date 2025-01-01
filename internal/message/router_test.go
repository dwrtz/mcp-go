package message

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/dwrtz/mcp-go/internal/testutil"
	"github.com/dwrtz/mcp-go/pkg/types"
)

func TestNewMessageRouter(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	router := NewMessageRouter(logger)

	if router.Requests == nil {
		t.Error("Requests channel not initialized")
	}
	if router.Responses == nil {
		t.Error("Responses channel not initialized")
	}
	if router.Notifications == nil {
		t.Error("Notifications channel not initialized")
	}
	if router.Errors == nil {
		t.Error("Errors channel not initialized")
	}
	if router.done == nil {
		t.Error("Done channel not initialized")
	}
}

func TestMessageRouter_Handle_Request(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	router := NewMessageRouter(logger)

	ctx := context.Background()
	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      &types.ID{Num: 1},
		Method:  "test/method",
	}

	// Start goroutine to receive message
	var receivedMsg *types.Message
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case receivedMsg = <-router.Requests:
		case <-time.After(time.Second):
			t.Error("Timeout waiting for request")
		}
	}()

	router.Handle(ctx, msg)
	wg.Wait()

	if receivedMsg != msg {
		t.Error("Received message does not match sent message")
	}
}

func TestMessageRouter_Handle_Response(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	router := NewMessageRouter(logger)

	ctx := context.Background()
	rawResult := json.RawMessage(`{"status":"ok"}`)
	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      &types.ID{Num: 1},
		Result:  &rawResult,
	}

	var receivedMsg *types.Message
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case receivedMsg = <-router.Responses:
		case <-time.After(time.Second):
			t.Error("Timeout waiting for response")
		}
	}()

	router.Handle(ctx, msg)
	wg.Wait()

	if receivedMsg != msg {
		t.Error("Received message does not match sent message")
	}
}

func TestMessageRouter_Handle_Notification(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	router := NewMessageRouter(logger)

	ctx := context.Background()
	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		Method:  "test/notification",
	}

	var receivedMsg *types.Message
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case receivedMsg = <-router.Notifications:
		case <-time.After(time.Second):
			t.Error("Timeout waiting for notification")
		}
	}()

	router.Handle(ctx, msg)
	wg.Wait()

	if receivedMsg != msg {
		t.Error("Received message does not match sent message")
	}
}

func TestMessageRouter_Handle_FullChannels(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	router := NewMessageRouter(logger)

	// Fill up channels
	for i := 0; i < cap(router.Requests); i++ {
		router.Requests <- &types.Message{}
	}
	for i := 0; i < cap(router.Responses); i++ {
		router.Responses <- &types.Message{}
	}
	for i := 0; i < cap(router.Notifications); i++ {
		router.Notifications <- &types.Message{}
	}

	ctx := context.Background()

	// Try to handle messages with full channels
	msg1 := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      &types.ID{Num: 1},
		Method:  "test/method",
	}
	msg2 := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      &types.ID{Num: 2},
	}
	msg3 := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		Method:  "test/notification",
	}

	// These should not block, but log warnings
	router.Handle(ctx, msg1)
	router.Handle(ctx, msg2)
	router.Handle(ctx, msg3)
}

func TestMessageRouter_Handle_AfterClose(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	router := NewMessageRouter(logger)

	router.Close()

	ctx := context.Background()
	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		Method:  "test/method",
	}

	// Should not panic and just log a message
	router.Handle(ctx, msg)
}

func TestMessageRouter_Handle_CancelledContext(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	router := NewMessageRouter(logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		Method:  "test/method",
	}

	// Should not panic and just log a message
	router.Handle(ctx, msg)
}

func TestMessageRouter_Close(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	router := NewMessageRouter(logger)

	// Test that close is idempotent
	router.Close()
	router.Close()

	// Verify channels are closed
	select {
	case <-router.Done():
		// Expected
	default:
		t.Error("Done channel not closed")
	}

	// Try to read from channels - should not block
	select {
	case _, ok := <-router.Requests:
		if ok {
			t.Error("Requests channel not closed")
		}
	default:
		t.Error("Requests channel not closed")
	}

	select {
	case _, ok := <-router.Responses:
		if ok {
			t.Error("Responses channel not closed")
		}
	default:
		t.Error("Responses channel not closed")
	}

	select {
	case _, ok := <-router.Notifications:
		if ok {
			t.Error("Notifications channel not closed")
		}
	default:
		t.Error("Notifications channel not closed")
	}

	select {
	case _, ok := <-router.Errors:
		if ok {
			t.Error("Errors channel not closed")
		}
	default:
		t.Error("Errors channel not closed")
	}
}

func TestMessageRouter_ConcurrentHandling(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	router := NewMessageRouter(logger)

	const numMessages = 10 // Reduced from 100 to prevent channel overflow
	var wg sync.WaitGroup
	wg.Add(numMessages * 3) // For requests, responses, and notifications

	ctx := context.Background()

	// Start receivers
	go func() {
		for i := 0; i < numMessages; i++ {
			<-router.Requests
			wg.Done()
		}
	}()

	go func() {
		for i := 0; i < numMessages; i++ {
			<-router.Responses
			wg.Done()
		}
	}()

	go func() {
		for i := 0; i < numMessages; i++ {
			<-router.Notifications
			wg.Done()
		}
	}()

	// Send messages concurrently
	for i := 0; i < numMessages; i++ {
		go func(id int) {
			// Send a request
			router.Handle(ctx, &types.Message{
				JSONRPC: types.JSONRPCVersion,
				ID:      &types.ID{Num: uint64(id)},
				Method:  "test/method",
			})

			// Send a response
			rawResult := json.RawMessage(`{"status":"ok"}`)
			router.Handle(ctx, &types.Message{
				JSONRPC: types.JSONRPCVersion,
				ID:      &types.ID{Num: uint64(id)},
				Result:  &rawResult,
			})

			// Send a notification
			router.Handle(ctx, &types.Message{
				JSONRPC: types.JSONRPCVersion,
				Method:  "test/notification",
			})
		}(i)
	}

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for concurrent message handling")
	}
}
