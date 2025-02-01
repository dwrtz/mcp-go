package roots

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/internal/mock"
	"github.com/dwrtz/mcp-go/internal/testutil"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

func setupTest(t *testing.T) (context.Context, *Client, *base.Base, func()) {
	logger := testutil.NewTestLogger(t)
	serverTransport, clientTransport := mock.NewMockPipeTransports(logger)
	baseServer := base.NewBase(serverTransport)
	baseClient := base.NewBase(clientTransport)

	initialRoots := []types.Root{
		{
			URI:  "file:///test/dir",
			Name: "Test Directory",
		},
	}
	rootsClient := NewClient(baseClient, initialRoots)

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

	return ctx, rootsClient, baseServer, cleanup
}

func TestClient_SetRoots(t *testing.T) {
	testCases := []struct {
		name    string
		roots   []types.Root
		wantErr bool
	}{
		{
			name: "valid roots",
			roots: []types.Root{
				{
					URI:  "file:///project/src",
					Name: "Source Directory",
				},
				{
					URI:  "file:///project/docs",
					Name: "Documentation",
				},
			},
			wantErr: false,
		},
		{
			name:    "empty roots list",
			roots:   []types.Root{},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, client, server, cleanup := setupTest(t)
			defer cleanup()

			// Channel to track notifications
			notificationReceived := make(chan struct{})

			// Register notification handler on server
			server.RegisterNotificationHandler(methods.RootsChanged, func(ctx context.Context, params json.RawMessage) {
				close(notificationReceived)
			})

			// Set roots
			if err := client.SetRoots(ctx, tc.roots); err != nil {
				if !tc.wantErr {
					t.Errorf("SetRoots() unexpected error: %v", err)
				}
				return
			} else if tc.wantErr {
				t.Error("SetRoots() expected error, got none")
				return
			}

			// Wait for notification
			select {
			case <-notificationReceived:
				// Success
			case <-time.After(time.Second):
				t.Error("Timeout waiting for roots changed notification")
				return
			}

			// Verify roots are set correctly by making a list request
			req := &types.ListRootsRequest{
				Method: methods.ListRoots,
			}
			resp, err := server.SendRequest(ctx, methods.ListRoots, req)
			if err != nil {
				t.Fatalf("Failed to list roots: %v", err)
			}

			var result types.ListRootsResult
			if err := json.Unmarshal(*resp.Result, &result); err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}

			// Compare roots
			if len(result.Roots) != len(tc.roots) {
				t.Errorf("Expected %d roots, got %d", len(tc.roots), len(result.Roots))
				return
			}

			for i, want := range tc.roots {
				got := result.Roots[i]
				if got.URI != want.URI {
					t.Errorf("Root %d URI mismatch: got %s, want %s", i, got.URI, want.URI)
				}
				if got.Name != want.Name {
					t.Errorf("Root %d Name mismatch: got %s, want %s", i, got.Name, want.Name)
				}
			}
		})
	}
}

func TestClient_HandleInvalidRoots(t *testing.T) {
	ctx, client, _, cleanup := setupTest(t)
	defer cleanup()

	// Try to set roots with invalid URI scheme
	invalidRoots := []types.Root{
		{
			URI:  "invalid:///path",
			Name: "Invalid Root",
		},
	}

	err := client.SetRoots(ctx, invalidRoots)
	if err == nil {
		t.Error("Expected error for invalid root URI, got none")
	} else if !strings.Contains(err.Error(), "root URI must start with file://") {
		t.Errorf("Expected error about file:// scheme, got: %v", err)
	}
}

func TestClient_ConcurrentRootUpdates(t *testing.T) {
	ctx, client, server, cleanup := setupTest(t)
	defer cleanup()

	notificationCount := 0
	var notificationMu sync.Mutex
	notificationChan := make(chan struct{})

	server.RegisterNotificationHandler(methods.RootsChanged, func(ctx context.Context, params json.RawMessage) {
		notificationMu.Lock()
		notificationCount++
		notificationMu.Unlock()
		notificationChan <- struct{}{}
	})

	// Make several concurrent root updates
	const numUpdates = 5
	errChan := make(chan error, numUpdates)

	for i := 0; i < numUpdates; i++ {
		go func(i int) {
			roots := []types.Root{
				{
					URI:  fmt.Sprintf("file:///project/dir%d", i),
					Name: fmt.Sprintf("Directory %d", i),
				},
			}
			errChan <- client.SetRoots(ctx, roots)
		}(i)
	}

	// Wait for all updates
	for i := 0; i < numUpdates; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Update %d failed: %v", i, err)
		}
		select {
		case <-notificationChan:
			// Success
		case <-time.After(time.Second):
			t.Errorf("Timeout waiting for notification %d", i)
		}
	}

	notificationMu.Lock()
	finalCount := notificationCount
	notificationMu.Unlock()

	if finalCount != numUpdates {
		t.Errorf("Expected %d notifications, got %d", numUpdates, finalCount)
	}
}
