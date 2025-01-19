package resources

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

func setupTest(t *testing.T) (context.Context, *ResourcesClient, *base.Server, func()) {
	logger := testutil.NewTestLogger(t)
	serverTransport, clientTransport := mock.NewMockPipeTransports(logger)
	baseServer := base.NewServer(serverTransport)
	baseClient := base.NewClient(clientTransport)

	resourcesClient := NewResourcesClient(baseClient)

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

	return ctx, resourcesClient, baseServer, cleanup
}

func TestResourcesClient_List(t *testing.T) {
	tests := []struct {
		name     string
		response *types.ListResourcesResult
		wantErr  bool
		errorMsg string
	}{
		{
			name: "successful resource listing",
			response: &types.ListResourcesResult{
				Resources: []types.Resource{
					{
						URI:      "file:///project/src/main.rs",
						Name:     "main.rs",
						MimeType: "text/x-rust",
					},
					{
						URI:      "file:///project/README.md",
						Name:     "README.md",
						MimeType: "text/markdown",
					},
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
			server.RegisterRequestHandler(methods.ListResources, func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				if tt.wantErr {
					return nil, types.NewError(types.InternalError, tt.errorMsg)
				}
				return tt.response, nil
			})

			// Make request
			resources, err := client.List(ctx)

			// Verify results
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(resources) != len(tt.response.Resources) {
					t.Errorf("Expected %d resources, got %d", len(tt.response.Resources), len(resources))
					return
				}

				for i, want := range tt.response.Resources {
					got := resources[i]
					if got.URI != want.URI {
						t.Errorf("Resource %d URI mismatch: want %s, got %s", i, want.URI, got.URI)
					}
					if got.Name != want.Name {
						t.Errorf("Resource %d Name mismatch: want %s, got %s", i, want.Name, got.Name)
					}
					if got.MimeType != want.MimeType {
						t.Errorf("Resource %d MimeType mismatch: want %s, got %s", i, want.MimeType, got.MimeType)
					}
				}
			}
		})
	}
}

func TestResourcesClient_Read(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		response *types.ReadResourceResult
		wantErr  bool
		errCode  int
		errMsg   string
	}{
		{
			name: "successful text resource read",
			uri:  "file:///project/README.md",
			response: &types.ReadResourceResult{
				Contents: []types.ResourceContent{
					types.TextResourceContents{
						ResourceContents: types.ResourceContents{
							URI:      "file:///project/README.md",
							MimeType: "text/markdown",
						},
						Text: "# Project\nThis is a test project.",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "resource not found",
			uri:     "file:///nonexistent",
			wantErr: true,
			errCode: types.InvalidParams,
			errMsg:  "resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, client, server, cleanup := setupTest(t)
			defer cleanup()

			// Register request handler
			server.RegisterRequestHandler(methods.ReadResource, func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				var req types.ReadResourceRequest
				if err := json.Unmarshal(params, &req); err != nil {
					return nil, err
				}

				if tt.wantErr {
					return nil, types.NewError(tt.errCode, tt.errMsg)
				}

				if req.URI != tt.uri {
					t.Errorf("Expected URI %s, got %s", tt.uri, req.URI)
				}

				return tt.response, nil
			})

			// Make request
			contents, err := client.Read(ctx, tt.uri)

			// Verify results
			if (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if mcpErr, ok := err.(*types.ErrorResponse); !ok {
					t.Errorf("Expected MCP error, got %T", err)
				} else {
					if mcpErr.Code != tt.errCode {
						t.Errorf("Expected error code %d, got %d", tt.errCode, mcpErr.Code)
					}
					if mcpErr.Message != tt.errMsg {
						t.Errorf("Expected error message %q, got %q", tt.errMsg, mcpErr.Message)
					}
				}
				return
			}

			if len(contents) != len(tt.response.Contents) {
				t.Errorf("Expected %d content items, got %d", len(tt.response.Contents), len(contents))
				return
			}

			for i, want := range tt.response.Contents {
				got := contents[i]
				switch w := want.(type) {
				case types.TextResourceContents:
					g, ok := got.(types.TextResourceContents)
					if !ok {
						t.Errorf("Content %d: expected TextResourceContents, got %T", i, got)
						continue
					}
					if g.URI != w.URI {
						t.Errorf("Content %d URI mismatch: want %s, got %s", i, w.URI, g.URI)
					}
					if g.Text != w.Text {
						t.Errorf("Content %d text mismatch: want %s, got %s", i, w.Text, g.Text)
					}
				}
			}
		})
	}
}

func TestResourcesClient_OnResourceUpdated(t *testing.T) {
	ctx, client, server, cleanup := setupTest(t)
	defer cleanup()

	// Channel to track callback invocation
	callbackCalled := make(chan string)

	// Register callback
	client.OnResourceUpdated(func(uri string) {
		callbackCalled <- uri
	})

	// Test URI
	testURI := "file:///project/src/main.rs"

	// Send notification with proper notification struct
	notification := types.ResourceUpdatedNotification{
		Method: methods.ResourceUpdated,
		URI:    testURI,
	}
	if err := server.SendNotification(ctx, methods.ResourceUpdated, notification); err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	// Wait for callback with timeout
	select {
	case receivedURI := <-callbackCalled:
		if receivedURI != testURI {
			t.Errorf("Expected URI %s, got %s", testURI, receivedURI)
		}
	case <-time.After(time.Second):
		t.Error("Callback not called within timeout")
	}
}

func TestResourcesClient_OnResourceListChanged(t *testing.T) {
	ctx, client, server, cleanup := setupTest(t)
	defer cleanup()

	// Channel to track callback invocation
	callbackCalled := make(chan struct{})

	// Register callback
	client.OnResourceListChanged(func() {
		close(callbackCalled)
	})

	// Send notification with empty struct as params
	notification := struct{}{}
	if err := server.SendNotification(ctx, methods.ResourceListChanged, notification); err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	// Wait for callback with timeout
	select {
	case <-callbackCalled:
		// Success
	case <-time.After(time.Second):
		t.Error("Callback not called within timeout")
	}
}
