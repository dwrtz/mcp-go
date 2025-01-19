package client

import (
	"testing"

	"github.com/dwrtz/mcp-go/internal/mock"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

func TestResourcesClient_List(t *testing.T) {
	t.Log("TestResourcesClient_List")
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

			// Create resources client
			client := NewResourcesClient(mockClient.BaseClient)

			// Handle the expected request
			done := mockClient.ExpectRequest(methods.ListResources, func(msg *types.Message) *types.Message {
				if tt.wantErr {
					return mockClient.CreateErrorResponse(msg, types.InternalError, tt.errorMsg, nil)
				}
				return mockClient.CreateSuccessResponse(msg, &types.ListResourcesResult{
					Resources: tt.resources,
				})
			})

			// Make the actual request
			resources, err := client.List(mockClient.Context)

			// Wait for mock handler to complete
			<-done

			// Verify results
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

			mockClient.AssertNoErrors(t)
		})
	}
}

func TestResourcesClient_Read(t *testing.T) {
	t.Log("TestResourcesClient_Read")
	tests := []struct {
		name     string
		uri      string
		contents []types.ResourceContent
		wantErr  bool
		errCode  int
		errMsg   string
	}{
		{
			name: "successful text resource read",
			uri:  "file:///project/README.md",
			contents: []types.ResourceContent{
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
			name:    "resource not found",
			uri:     "file:///nonexistent",
			wantErr: true,
			errCode: types.InvalidParams,
			errMsg:  "resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mock.NewMockClient(t)
			defer mockClient.Close()

			client := NewResourcesClient(mockClient.BaseClient)

			done := mockClient.ExpectRequest(methods.ReadResource, func(msg *types.Message) *types.Message {
				if tt.wantErr {
					return mockClient.CreateErrorResponse(msg, tt.errCode, tt.errMsg, nil)
				}
				return mockClient.CreateSuccessResponse(msg, &types.ReadResourceResult{
					Contents: tt.contents,
				})
			})

			contents, err := client.Read(mockClient.Context, tt.uri)
			<-done

			if (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(contents) != len(tt.contents) {
					t.Errorf("Expected %d content items, got %d", len(tt.contents), len(contents))
				}

				for i, want := range tt.contents {
					wantContents, ok := want.(types.TextResourceContents)
					if !ok {
						t.Errorf("Expected TextResourceContents at index %d", i)
						continue
					}

					gotContents, ok := contents[i].(types.TextResourceContents)
					if !ok {
						t.Errorf("Got unexpected type at index %d: %T", i, contents[i])
						continue
					}

					if gotContents.URI != wantContents.URI {
						t.Errorf("Content %d URI mismatch: want %s, got %s", i, wantContents.URI, gotContents.URI)
					}
					if gotContents.Text != wantContents.Text {
						t.Errorf("Content %d text mismatch: want %s, got %s", i, wantContents.Text, gotContents.Text)
					}
				}
			}

			mockClient.AssertNoErrors(t)
		})
	}
}

func TestResourcesClient_OnResourceUpdated(t *testing.T) {
	t.Log("TestResourcesClient_OnResourceUpdated")
	mockClient := mock.NewMockClient(t)
	defer mockClient.Close()

	client := NewResourcesClient(mockClient.BaseClient)

	// Channel to track callback invocation
	callbackInvoked := make(chan string)

	// Register callback
	client.OnResourceUpdated(func(uri string) {
		callbackInvoked <- uri
	})

	// Test URI
	testURI := "file:///project/src/main.rs"

	// Test notification handling
	err := mockClient.SimulateNotification(methods.ResourceUpdated, &types.ResourceUpdatedNotification{
		Method: methods.ResourceUpdated,
		URI:    testURI,
	})
	if err != nil {
		t.Fatalf("Failed to simulate notification: %v", err)
	}

	// Wait for callback with timeout
	if err := mockClient.WaitForCallback(func(done chan<- struct{}) {
		select {
		case receivedURI := <-callbackInvoked:
			if receivedURI != testURI {
				t.Errorf("Expected URI %s, got %s", testURI, receivedURI)
			}
			close(done)
		case <-mockClient.Context.Done():
			t.Error("Context cancelled while waiting for callback")
		}
	}); err != nil {
		t.Errorf("Error waiting for callback: %v", err)
	}

	mockClient.AssertNoErrors(t)
}
