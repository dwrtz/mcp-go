package client

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

func setupTestRootsClient(t *testing.T) (*RootsClient, *mock.MockTransport, context.Context, context.CancelFunc) {
	logger := testutil.NewTestLogger(t)
	mockTransport := mock.NewMockTransport(logger)
	baseClient := base.NewClient(mockTransport)
	client := NewRootsClient(baseClient)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// Start the client and transport
	err := mockTransport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}

	err = baseClient.Start(ctx)
	if err != nil {
		cancel()
		t.Fatalf("Failed to start client: %v", err)
	}

	return client, mockTransport, ctx, cancel
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
			name:    "empty root list",
			roots:   []types.Root{},
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
			client, mockTransport, ctx, cancel := setupTestRootsClient(t)
			defer cancel()

			// Clear any previous messages
			mockTransport.ClearSentMessages()

			// Channel to coordinate test completion
			done := make(chan struct{})

			// Handle mock responses
			go func() {
				defer close(done)
				select {
				case msg := <-mockTransport.GetRouter().Requests:
					if msg == nil || msg.ID == nil {
						t.Error("Received nil message or message ID")
						return
					}
					if msg.Method != methods.ListRoots {
						t.Errorf("Expected method %s, got %s", methods.ListRoots, msg.Method)
					}

					response := &types.Message{
						JSONRPC: types.JSONRPCVersion,
						ID:      msg.ID,
					}

					if tt.wantErr {
						response.Error = &types.ErrorResponse{
							Code:    types.InternalError,
							Message: tt.errorMsg,
						}
						response.Result = nil // Ensure result is nil when there's an error
					} else {
						result := &types.ListRootsResult{
							Roots: tt.roots,
						}
						data, err := testutil.MarshalResult(result)
						if err != nil {
							t.Errorf("Failed to marshal result: %v", err)
							return
						}
						response.Result = data
					}

					mockTransport.SimulateReceive(ctx, response)

				case <-ctx.Done():
					t.Error("Context cancelled while waiting for request")
					return
				}
			}()

			// Make the request
			roots, err := client.List(ctx)

			// Wait for mock handler to complete
			<-done

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify results
				if len(roots) != len(tt.roots) {
					t.Errorf("Expected %d roots, got %d", len(tt.roots), len(roots))
				}

				// Compare each root
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
	client, mockTransport, ctx, cancel := setupTestRootsClient(t)
	defer cancel()

	// Channel to track callback invocation
	callbackInvoked := make(chan struct{})

	// Register callback
	client.OnRootsChanged(func() {
		close(callbackInvoked)
	})

	// Simulate server sending a notification with empty params
	notification := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		Method:  methods.RootsChanged,
		Params:  &json.RawMessage{'{', '}'}, // Empty JSON object as params
	}
	mockTransport.SimulateReceive(ctx, notification)

	// Wait for callback with timeout
	select {
	case <-callbackInvoked:
		// Success - callback was invoked
	case <-time.After(time.Second):
		t.Error("Timeout waiting for roots changed callback")
	case <-ctx.Done():
		t.Error("Context cancelled while waiting for callback")
	}
}
