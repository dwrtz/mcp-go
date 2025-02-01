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

func setupTest(t *testing.T) (context.Context, *Server, *base.Base, func()) {
	logger := testutil.NewTestLogger(t)
	serverTransport, clientTransport := mock.NewMockPipeTransports(logger)
	baseServer := base.NewBase(serverTransport)
	baseClient := base.NewBase(clientTransport)

	initialResources := []types.Resource{
		{
			URI:      "file:///test.txt",
			Name:     "Test File",
			MimeType: "text/plain",
		},
	}

	initialTemplates := []types.ResourceTemplate{
		{
			URITemplate: "file:///test/{name}.txt",
			Name:        "Text File Template",
			MimeType:    "text/plain",
		},
	}

	resourcesServer := NewServer(baseServer, initialResources, initialTemplates)

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

	return ctx, resourcesServer, baseClient, cleanup
}

func TestServer_ListResources(t *testing.T) {
	tests := []struct {
		name      string
		resources []types.Resource
		wantErr   bool
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, server, client, cleanup := setupTest(t)
			defer cleanup()

			// Set resources on server
			if err := server.SetResources(ctx, tt.resources); err != nil {
				t.Fatalf("Failed to set resources: %v", err)
			}

			// Send list request from client
			req := &types.ListResourcesRequest{
				Method: methods.ListResources,
			}
			resp, err := client.SendRequest(ctx, methods.ListResources, req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListResources error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				var result types.ListResourcesResult
				if err := json.Unmarshal(*resp.Result, &result); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if len(result.Resources) != len(tt.resources) {
					t.Errorf("Expected %d resources, got %d", len(tt.resources), len(result.Resources))
					return
				}

				for i, want := range tt.resources {
					got := result.Resources[i]
					if got.URI != want.URI {
						t.Errorf("Resource %d URI mismatch: got %s, want %s", i, got.URI, want.URI)
					}
					if got.Name != want.Name {
						t.Errorf("Resource %d Name mismatch: got %s, want %s", i, got.Name, want.Name)
					}
					if got.MimeType != want.MimeType {
						t.Errorf("Resource %d MimeType mismatch: got %s, want %s", i, got.MimeType, want.MimeType)
					}
				}
			}
		})
	}
}

func TestServer_ReadResource(t *testing.T) {
	tests := []struct {
		name          string
		uri           string
		contentResult []types.ResourceContent
		wantErr       bool
		errCode       int
		errMsg        string
	}{
		{
			name: "read text resource",
			uri:  "file:///test.txt",
			contentResult: []types.ResourceContent{
				types.TextResourceContents{
					ResourceContents: types.ResourceContents{
						URI:      "file:///test.txt",
						MimeType: "text/plain",
					},
					Text: "Hello, World!",
				},
			},
			wantErr: false,
		},
		{
			name: "read binary resource",
			uri:  "file:///test.bin",
			contentResult: []types.ResourceContent{
				types.BlobResourceContents{
					ResourceContents: types.ResourceContents{
						URI:      "file:///test.bin",
						MimeType: "application/octet-stream",
					},
					Blob: "SGVsbG8sIFdvcmxkIQ==", // base64 encoded "Hello, World!"
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
			ctx, server, client, cleanup := setupTest(t)
			defer cleanup()

			// Register content handler
			server.RegisterContentHandler("file://", func(ctx context.Context, uri string) ([]types.ResourceContent, error) {
				if tt.wantErr {
					return nil, types.NewError(tt.errCode, tt.errMsg)
				}
				return tt.contentResult, nil
			})

			// Send read request from client
			req := &types.ReadResourceRequest{
				Method: methods.ReadResource,
				URI:    tt.uri,
			}
			resp, err := client.SendRequest(ctx, methods.ReadResource, req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadResource error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if mcpErr, ok := err.(*types.ErrorResponse); ok {
					if mcpErr.Code != tt.errCode {
						t.Errorf("Expected error code %d, got %d", tt.errCode, mcpErr.Code)
					}
					if mcpErr.Message != tt.errMsg {
						t.Errorf("Expected error message %q, got %q", tt.errMsg, mcpErr.Message)
					}
				} else {
					t.Errorf("Expected MCP error, got %T", err)
				}
				return
			}

			var result types.ReadResourceResult
			if err := json.Unmarshal(*resp.Result, &result); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if len(result.Contents) != len(tt.contentResult) {
				t.Errorf("Expected %d content items, got %d", len(tt.contentResult), len(result.Contents))
				return
			}

			for i, want := range tt.contentResult {
				got := result.Contents[i]
				switch w := want.(type) {
				case types.TextResourceContents:
					g, ok := got.(types.TextResourceContents)
					if !ok {
						t.Errorf("Content %d: expected TextResourceContents, got %T", i, got)
						continue
					}
					if g.URI != w.URI {
						t.Errorf("Content %d URI mismatch: got %s, want %s", i, g.URI, w.URI)
					}
					if g.Text != w.Text {
						t.Errorf("Content %d text mismatch: got %s, want %s", i, g.Text, w.Text)
					}
				case types.BlobResourceContents:
					g, ok := got.(types.BlobResourceContents)
					if !ok {
						t.Errorf("Content %d: expected BlobResourceContents, got %T", i, got)
						continue
					}
					if g.URI != w.URI {
						t.Errorf("Content %d URI mismatch: got %s, want %s", i, g.URI, w.URI)
					}
					if g.Blob != w.Blob {
						t.Errorf("Content %d blob mismatch: got %s, want %s", i, g.Blob, w.Blob)
					}
				default:
					t.Errorf("Unknown content type: %T", want)
				}
			}
		})
	}
}

func TestServer_ResourceNotifications(t *testing.T) {
	ctx, server, client, cleanup := setupTest(t)
	defer cleanup()

	// Channel to track notification receipt
	notificationReceived := make(chan string)

	// Register notification handler on client
	client.RegisterNotificationHandler(methods.ResourceUpdated, func(ctx context.Context, params json.RawMessage) {
		var notif types.ResourceUpdatedNotification
		if err := json.Unmarshal(params, &notif); err != nil {
			t.Errorf("Failed to unmarshal notification: %v", err)
			return
		}
		notificationReceived <- notif.URI
	})

	// Subscribe to a resource
	subscribeReq := &types.SubscribeRequest{
		Method: methods.SubscribeResource,
		URI:    "file:///test.txt",
	}
	_, err := client.SendRequest(ctx, methods.SubscribeResource, subscribeReq)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Notify about resource update
	if err := server.NotifyResourceUpdated(ctx, "file:///test.txt"); err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	// Wait for notification with timeout
	select {
	case uri := <-notificationReceived:
		if uri != "file:///test.txt" {
			t.Errorf("Expected notification for file:///test.txt, got %s", uri)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for notification")
	}

	// Unsubscribe
	unsubscribeReq := &types.UnsubscribeRequest{
		Method: methods.UnsubscribeResource,
		URI:    "file:///test.txt",
	}
	_, err = client.SendRequest(ctx, methods.UnsubscribeResource, unsubscribeReq)
	if err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}

	// Verify no more notifications are received
	if err := server.NotifyResourceUpdated(ctx, "file:///test.txt"); err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	select {
	case uri := <-notificationReceived:
		t.Errorf("Received unexpected notification for %s after unsubscribe", uri)
	case <-time.After(100 * time.Millisecond):
		// Success - no notification received
	}
}

func TestServer_ListTemplates(t *testing.T) {
	tests := []struct {
		name      string
		templates []types.ResourceTemplate
		wantErr   bool
	}{
		{
			name: "successful template listing",
			templates: []types.ResourceTemplate{
				{
					URITemplate: "file:///project/{name}.rs",
					Name:        "Rust Source File",
					MimeType:    "text/x-rust",
				},
				{
					URITemplate: "file:///project/{name}.md",
					Name:        "Markdown Document",
					MimeType:    "text/markdown",
				},
			},
			wantErr: false,
		},
		{
			name:      "empty template list",
			templates: []types.ResourceTemplate{},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, server, client, cleanup := setupTest(t)
			defer cleanup()

			// Set templates on server
			server.SetTemplates(ctx, tt.templates)

			// Send list request from client
			req := &types.ListResourceTemplatesRequest{
				Method: methods.ListResourceTemplates,
			}
			resp, err := client.SendRequest(ctx, methods.ListResourceTemplates, req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListResourceTemplates error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				var result types.ListResourceTemplatesResult
				if err := json.Unmarshal(*resp.Result, &result); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if len(result.ResourceTemplates) != len(tt.templates) {
					t.Errorf("Expected %d templates, got %d", len(tt.templates), len(result.ResourceTemplates))
					return
				}

				for i, want := range tt.templates {
					got := result.ResourceTemplates[i]
					if got.URITemplate != want.URITemplate {
						t.Errorf("Template %d URITemplate mismatch: got %s, want %s", i, got.URITemplate, want.URITemplate)
					}
					if got.Name != want.Name {
						t.Errorf("Template %d Name mismatch: got %s, want %s", i, got.Name, want.Name)
					}
					if got.MimeType != want.MimeType {
						t.Errorf("Template %d MimeType mismatch: got %s, want %s", i, got.MimeType, want.MimeType)
					}
				}
			}
		})
	}
}
