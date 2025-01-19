package roots

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/internal/mock"
	"github.com/dwrtz/mcp-go/internal/testutil"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

func setupTest(t *testing.T) (context.Context, *RootsServer, *base.Client, func()) {
	logger := testutil.NewTestLogger(t)
	serverTransport, clientTransport := mock.NewMockPipeTransports(logger)

	// Create base server and client
	baseServer := base.NewServer(serverTransport)
	baseClient := base.NewClient(clientTransport)

	// Create roots server
	rootsServer := NewRootsServer(baseServer)

	ctx := context.Background()
	if err := baseServer.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	if err := baseClient.Start(ctx); err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}

	cleanup := func() {
		baseClient.Close()
		baseServer.Close()
	}

	return ctx, rootsServer, baseClient, cleanup
}

func TestRootsServer_SetRoots(t *testing.T) {
	tests := []struct {
		name    string
		roots   []types.Root
		wantErr bool
	}{
		{
			name: "valid roots",
			roots: []types.Root{
				{
					URI:  "file:///project/src",
					Name: "Source Code",
				},
				{
					URI:  "file:///project/docs",
					Name: "Documentation",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid root URI",
			roots: []types.Root{
				{
					URI:  "invalid://path",
					Name: "Invalid",
				},
			},
			wantErr: true,
		},
		{
			name:    "empty roots list",
			roots:   []types.Root{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, server, _, cleanup := setupTest(t)
			defer cleanup()

			err := server.SetRoots(ctx, tt.roots)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetRoots() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRootsServer_ListRoots(t *testing.T) {
	ctx, server, client, cleanup := setupTest(t)
	defer cleanup()

	// Set up test roots
	testRoots := []types.Root{
		{
			URI:  "file:///project/src",
			Name: "Source Code",
		},
		{
			URI:  "file:///project/docs",
			Name: "Documentation",
		},
	}

	// Set the roots
	if err := server.SetRoots(ctx, testRoots); err != nil {
		t.Fatalf("Failed to set roots: %v", err)
	}

	// Send list roots request
	req := &types.ListRootsRequest{
		Method: methods.ListRoots,
	}

	resp, err := client.SendRequest(ctx, methods.ListRoots, req)
	if err != nil {
		t.Fatalf("ListRoots request failed: %v", err)
	}

	// Parse response
	var result types.ListRootsResult
	if err := json.Unmarshal(*resp.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify results
	if len(result.Roots) != len(testRoots) {
		t.Errorf("Expected %d roots, got %d", len(testRoots), len(result.Roots))
	}

	for i, want := range testRoots {
		if i >= len(result.Roots) {
			t.Errorf("Missing root at index %d", i)
			continue
		}
		got := result.Roots[i]
		if got.URI != want.URI {
			t.Errorf("Root %d URI mismatch: got %s, want %s", i, got.URI, want.URI)
		}
		if got.Name != want.Name {
			t.Errorf("Root %d Name mismatch: got %s, want %s", i, got.Name, want.Name)
		}
	}
}

func TestRootsServer_RootsChangedNotification(t *testing.T) {
	ctx, server, client, cleanup := setupTest(t)
	defer cleanup()

	// Channel to track notification
	notificationReceived := make(chan struct{})

	// Register notification handler
	client.RegisterNotificationHandler(methods.RootsChanged, func(ctx context.Context, params json.RawMessage) {
		close(notificationReceived)
	})

	// Set roots which should trigger notification
	testRoots := []types.Root{
		{
			URI:  "file:///project/src",
			Name: "Source Code",
		},
	}

	if err := server.SetRoots(ctx, testRoots); err != nil {
		t.Fatalf("Failed to set roots: %v", err)
	}

	// Wait for notification with timeout
	select {
	case <-notificationReceived:
		// Success
	case <-time.After(time.Second):
		t.Error("Timeout waiting for roots changed notification")
	}
}
