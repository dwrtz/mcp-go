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

func setupTestResourcesClient(t *testing.T) (*ResourcesClient, *mock.MockTransport, context.Context, context.CancelFunc) {
	logger := testutil.NewTestLogger(t)
	mockTransport := mock.NewMockTransport(logger)
	baseClient := base.NewClient(mockTransport)
	client := NewResourcesClient(baseClient)

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

func TestResourcesClient_List(t *testing.T) {
	tests := []struct {
		name      string
		resources []types.Resource
		wantErr   bool
		errorMsg  string
	}{
		{
			name: "successful resource listing",
			resources: []types.Resource{
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
			wantErr: false,
		},
		{
			name:      "empty resource list",
			resources: []types.Resource{},
			wantErr:   false,
		},
		{
			name:     "server error",
			wantErr:  true,
			errorMsg: "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockTransport, ctx, cancel := setupTestResourcesClient(t)
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
					if msg.Method != methods.ListResources {
						t.Errorf("Expected method %s, got %s", methods.ListResources, msg.Method)
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
					} else {
						result := &types.ListResourcesResult{
							Resources: tt.resources,
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
			resources, err := client.List(ctx)

			// Wait for mock handler to complete
			<-done

			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(resources) != len(tt.resources) {
					t.Errorf("Expected %d resources, got %d", len(tt.resources), len(resources))
				}

				for i, want := range tt.resources {
					if i >= len(resources) {
						t.Errorf("Missing resource at index %d", i)
						continue
					}
					if resources[i].URI != want.URI {
						t.Errorf("Resource %d URI mismatch: want %s, got %s", i, want.URI, resources[i].URI)
					}
					if resources[i].Name != want.Name {
						t.Errorf("Resource %d Name mismatch: want %s, got %s", i, want.Name, resources[i].Name)
					}
					if resources[i].MimeType != want.MimeType {
						t.Errorf("Resource %d MimeType mismatch: want %s, got %s", i, want.MimeType, resources[i].MimeType)
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
		contents []interface{}
		wantErr  bool
		errorMsg string
	}{
		{
			name: "successful text resource read",
			uri:  "file:///project/README.md",
			contents: []interface{}{
				types.TextResourceContents{
					ResourceContents: types.ResourceContents{
						URI:      "file:///project/README.md",
						MimeType: "text/markdown",
					},
					Text: "# Project\nThis is a test project.",
				},
			},
			wantErr: false,
		},
		{
			name:     "resource not found",
			uri:      "file:///nonexistent",
			wantErr:  true,
			errorMsg: "resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockTransport, ctx, cancel := setupTestResourcesClient(t)
			defer cancel()

			mockTransport.ClearSentMessages()

			done := make(chan struct{})

			go func() {
				defer close(done)
				select {
				case msg := <-mockTransport.GetRouter().Requests:
					if msg.Method != methods.ReadResource {
						t.Errorf("Expected method %s, got %s", methods.ReadResource, msg.Method)
					}

					// Verify request parameters
					var req types.ReadResourceRequest
					if err := json.Unmarshal(*msg.Params, &req); err != nil {
						t.Errorf("Failed to unmarshal request params: %v", err)
						return
					}
					if req.URI != tt.uri {
						t.Errorf("Expected URI %s, got %s", tt.uri, req.URI)
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
					} else {
						result := &types.ReadResourceResult{
							Contents: tt.contents,
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

			contents, err := client.Read(ctx, tt.uri)
			<-done

			if (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(contents) != len(tt.contents) {
					t.Errorf("Expected %d content items, got %d", len(tt.contents), len(contents))
				}

				// Compare contents (this is a simplified comparison)
				for i, want := range tt.contents {
					if i >= len(contents) {
						t.Errorf("Missing content at index %d", i)
						continue
					}
					wantJSON, _ := json.Marshal(want)
					gotJSON, _ := json.Marshal(contents[i])
					wantRaw := json.RawMessage(wantJSON)
					gotRaw := json.RawMessage(gotJSON)
					if !testutil.JSONEqual(t, &wantRaw, &gotRaw) {
						t.Errorf("Content %d mismatch:\nwant: %s\ngot:  %s", i, wantJSON, gotJSON)
					}
				}
			}
		})
	}
}

func TestResourcesClient_Subscribe(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		wantErr  bool
		errorMsg string
	}{
		{
			name:    "successful subscription",
			uri:     "file:///project/src/main.rs",
			wantErr: false,
		},
		{
			name:     "invalid resource",
			uri:      "invalid://uri",
			wantErr:  true,
			errorMsg: "invalid resource URI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockTransport, ctx, cancel := setupTestResourcesClient(t)
			defer cancel()

			mockTransport.ClearSentMessages()
			done := make(chan struct{})

			go func() {
				defer close(done)
				select {
				case msg := <-mockTransport.GetRouter().Requests:
					if msg.Method != methods.SubscribeResource {
						t.Errorf("Expected method %s, got %s", methods.SubscribeResource, msg.Method)
					}

					var req types.SubscribeRequest
					if err := json.Unmarshal(*msg.Params, &req); err != nil {
						t.Errorf("Failed to unmarshal request params: %v", err)
						return
					}
					if req.URI != tt.uri {
						t.Errorf("Expected URI %s, got %s", tt.uri, req.URI)
					}

					response := &types.Message{
						JSONRPC: types.JSONRPCVersion,
						ID:      msg.ID,
					}

					if tt.wantErr {
						response.Error = &types.ErrorResponse{
							Code:    types.InvalidParams,
							Message: tt.errorMsg,
						}
					} else {
						response.Result = &json.RawMessage{'{', '}'}
					}

					mockTransport.SimulateReceive(ctx, response)

				case <-ctx.Done():
					t.Error("Context cancelled while waiting for request")
					return
				}
			}()

			err := client.Subscribe(ctx, tt.uri)
			<-done

			if (err != nil) != tt.wantErr {
				t.Errorf("Subscribe() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResourcesClient_ResourceUpdated(t *testing.T) {
	client, mockTransport, ctx, cancel := setupTestResourcesClient(t)
	defer cancel()

	// Channel to track callback invocation
	callbackInvoked := make(chan string)

	// Register callback
	client.OnResourceUpdated(func(uri string) {
		callbackInvoked <- uri
	})

	// Test URI
	testURI := "file:///project/src/main.rs"

	// Simulate server sending a resource updated notification
	params := map[string]string{"uri": testURI}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal params: %v", err)
	}
	rawMessage := json.RawMessage(paramsJSON)

	notification := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		Method:  methods.ResourceUpdated,
		Params:  &rawMessage,
	}
	mockTransport.SimulateReceive(ctx, notification)

	// Wait for callback with timeout
	select {
	case receivedURI := <-callbackInvoked:
		if receivedURI != testURI {
			t.Errorf("Expected URI %s, got %s", testURI, receivedURI)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for resource updated callback")
	case <-ctx.Done():
		t.Error("Context cancelled while waiting for callback")
	}
}

func TestResourcesClient_ListChanged(t *testing.T) {
	client, mockTransport, ctx, cancel := setupTestResourcesClient(t)
	defer cancel()

	// Channel to track callback invocation
	callbackInvoked := make(chan struct{})

	// Register callback
	client.OnResourceListChanged(func() {
		close(callbackInvoked)
	})

	// Simulate server sending a list changed notification
	params := struct{}{}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal params: %v", err)
	}
	rawMessage := json.RawMessage(paramsJSON)

	notification := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		Method:  methods.ResourceListChanged,
		Params:  &rawMessage,
	}
	mockTransport.SimulateReceive(ctx, notification)

	// Wait for callback with timeout
	select {
	case <-callbackInvoked:
		// Success - callback was invoked
	case <-time.After(time.Second):
		t.Error("Timeout waiting for list changed callback")
	case <-ctx.Done():
		t.Error("Context cancelled while waiting for callback")
	}
}
