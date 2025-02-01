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

func setupTest(t *testing.T) (context.Context, *RootsServer, *base.Base, func()) {
	logger := testutil.NewTestLogger(t)
	serverTransport, clientTransport := mock.NewMockPipeTransports(logger)
	baseServer := base.NewBase(serverTransport)
	baseClient := base.NewBase(clientTransport)

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

func TestRootsServer_List(t *testing.T) {
	tests := []struct {
		name           string
		clientResponse *types.ListRootsResult
		wantErr        bool
	}{
		{
			name: "successful roots listing",
			clientResponse: &types.ListRootsResult{
				Roots: []types.Root{
					{
						URI:  "file:///project/src",
						Name: "Source Directory",
					},
					{
						URI:  "file:///project/docs",
						Name: "Documentation",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty roots list",
			clientResponse: &types.ListRootsResult{
				Roots: []types.Root{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, server, clientBase, cleanup := setupTest(t)
			defer cleanup()

			// Register handler on client side to respond to server's list request
			clientBase.RegisterRequestHandler(methods.ListRoots, func(ctx context.Context, params *json.RawMessage) (interface{}, error) {
				return tt.clientResponse, nil
			})

			roots, err := server.ListRoots(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(roots) != len(tt.clientResponse.Roots) {
					t.Errorf("Expected %d roots, got %d", len(tt.clientResponse.Roots), len(roots))
					return
				}

				for i, want := range tt.clientResponse.Roots {
					got := roots[i]
					if got.URI != want.URI {
						t.Errorf("Root %d URI mismatch: got %s, want %s", i, got.URI, want.URI)
					}
					if got.Name != want.Name {
						t.Errorf("Root %d Name mismatch: got %s, want %s", i, got.Name, want.Name)
					}
				}
			}
		})
	}
}

func TestRootsServer_OnRootsChanged(t *testing.T) {
	ctx, server, clientBase, cleanup := setupTest(t)
	defer cleanup()

	// Channel to track notification reception
	notificationReceived := make(chan struct{})

	// Register callback
	server.OnRootsChanged(func() {
		close(notificationReceived)
	})

	// Send root change notification from client to server
	if err := clientBase.SendNotification(ctx, methods.RootsChanged, nil); err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	// Wait for notification with timeout
	select {
	case <-notificationReceived:
		// Success - notification was received and callback triggered
	case <-time.After(time.Second):
		t.Error("Timeout waiting for roots changed notification")
	}
}

func TestRootsServer_InvalidRoots(t *testing.T) {
	ctx, server, clientBase, cleanup := setupTest(t)
	defer cleanup()

	// Setup client to return invalid roots
	clientBase.RegisterRequestHandler(methods.ListRoots, func(ctx context.Context, params *json.RawMessage) (interface{}, error) {
		return &types.ListRootsResult{
			Roots: []types.Root{
				{
					URI:  "invalid://not-file-uri",
					Name: "Invalid Root",
				},
			},
		}, nil
	})

	// Test listing
	roots, err := server.ListRoots(ctx)
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}

	// Even though the URI is invalid according to spec, the server should still return it
	// as validation is the client's responsibility
	if len(roots) != 1 {
		t.Fatalf("Expected 1 root, got %d", len(roots))
	}

	// Validate the returned root
	if roots[0].URI != "invalid://not-file-uri" {
		t.Errorf("Expected invalid URI, got %s", roots[0].URI)
	}

	// Now validate the root using Root.Validate()
	if err := roots[0].Validate(); err == nil {
		t.Error("Expected validation error for invalid URI scheme")
	}
}

func TestRootsServer_ListErrorHandling(t *testing.T) {
	ctx, server, clientBase, cleanup := setupTest(t)
	defer cleanup()

	// Setup client to return an error
	clientBase.RegisterRequestHandler(methods.ListRoots, func(ctx context.Context, params *json.RawMessage) (interface{}, error) {
		return nil, types.NewError(types.InternalError, "internal server error")
	})

	// Test listing
	_, err := server.ListRoots(ctx)

	// Verify error
	if err == nil {
		t.Fatal("Expected error from ListRoots(), got nil")
	}

	// Check if it's the correct error
	mcpErr, ok := err.(*types.ErrorResponse)
	if !ok {
		t.Fatalf("Expected *types.ErrorResponse, got %T", err)
	}

	if mcpErr.Code != types.InternalError {
		t.Errorf("Expected error code %d, got %d", types.InternalError, mcpErr.Code)
	}
	if mcpErr.Message != "internal server error" {
		t.Errorf("Expected error message %q, got %q", "internal server error", mcpErr.Message)
	}
}
