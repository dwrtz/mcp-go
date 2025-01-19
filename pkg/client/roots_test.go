package client

import (
	"testing"

	"github.com/dwrtz/mcp-go/internal/mock"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

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
			// Create mock client with all dependencies
			mockClient := mock.NewMockClient(t)
			defer mockClient.Close()

			// Create roots client
			client := NewRootsClient(mockClient.BaseClient)

			// Handle the expected request
			done := mockClient.ExpectRequest(methods.ListRoots, func(msg *types.Message) *types.Message {
				if tt.wantErr {
					return mockClient.CreateErrorResponse(msg, types.InternalError, tt.errorMsg, nil)
				}
				return mockClient.CreateSuccessResponse(msg, &types.ListRootsResult{
					Roots: tt.roots,
				})
			})

			// Make the actual request
			roots, err := client.List(mockClient.Context)

			// Wait for mock handler to complete
			<-done

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

			mockClient.AssertNoErrors(t)
		})
	}
}

func TestRootsClient_OnRootsChanged(t *testing.T) {
	mockClient := mock.NewMockClient(t)
	defer mockClient.Close()

	client := NewRootsClient(mockClient.BaseClient)

	// Channel to track callback invocation
	callbackInvoked := make(chan struct{})

	// Register callback
	client.OnRootsChanged(func() {
		close(callbackInvoked)
	})

	// Test notification handling
	err := mockClient.SimulateNotification(methods.RootsChanged, struct{}{})
	if err != nil {
		t.Fatalf("Failed to simulate notification: %v", err)
	}

	// Wait for callback with timeout
	if err := mockClient.WaitForCallback(func(done chan<- struct{}) {
		select {
		case <-callbackInvoked:
			close(done)
		case <-mockClient.Context.Done():
			t.Error("Context cancelled while waiting for callback")
		}
	}); err != nil {
		t.Errorf("Error waiting for callback: %v", err)
	}

	mockClient.AssertNoErrors(t)
}
