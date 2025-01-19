package roots

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/internal/mock"
	"github.com/dwrtz/mcp-go/internal/testutil"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

func setupTest(t *testing.T) (context.Context, *RootsClient, *base.Server, func()) {
	logger := testutil.NewTestLogger(t)
	transport := mock.NewMockTransport(logger)

	baseClient := base.NewClient(transport)
	rootsClient := NewRootsClient(baseClient)

	baseServer := base.NewServer(transport)

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

func TestRootsClient_List(t *testing.T) {
	tests := []struct {
		name     string
		roots    []types.Root
		wantErr  bool
		errorMsg string
	}{
		{
			name: "successful root listing",
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
			name:     "server error",
			wantErr:  true,
			errorMsg: "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, client, server, cleanup := setupTest(t)
			defer cleanup()

			// Register request handler
			server.RegisterRequestHandler(methods.ListRoots, func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				if tt.wantErr {
					return nil, types.NewError(types.InternalError, tt.errorMsg)
				}
				return &types.ListRootsResult{
					Roots: tt.roots,
				}, nil
			})

			// Make request
			roots, err := client.List(ctx)

			// Verify results
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(roots) != len(tt.roots) {
					t.Errorf("Expected %d roots, got %d", len(tt.roots), len(roots))
				}

				for i, want := range tt.roots {
					if i >= len(roots) {
						t.Errorf("Missing root at index %d", i)
						continue
					}
					if roots[i].URI != want.URI {
						t.Errorf("Root %d URI mismatch: want %s, got %s", i, want.URI, roots[i].URI)
					}
					if roots[i].Name != want.Name {
						t.Errorf("Root %d Name mismatch: want %s, got %s", i, want.Name, roots[i].Name)
					}
				}
			}
		})
	}
}

func TestRootsClient_OnRootsChanged(t *testing.T) {
	ctx, client, server, cleanup := setupTest(t)
	defer cleanup()

	// Channel to track callback invocation
	callbackInvoked := make(chan struct{})

	// Register callback
	client.OnRootsChanged(func() {
		close(callbackInvoked)
	})

	// Send notification from server
	err := server.SendNotification(ctx, methods.RootsChanged, struct{}{})
	if err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	// Wait for callback with timeout
	select {
	case <-callbackInvoked:
		// Success
	case <-ctx.Done():
		t.Error("Context cancelled while waiting for callback")
	}
}
