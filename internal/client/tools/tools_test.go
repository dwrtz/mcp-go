package tools

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

func setupTest(t *testing.T) (context.Context, *Client, *base.Base, func()) {
	logger := testutil.NewTestLogger(t)
	serverTransport, clientTransport := mock.NewMockPipeTransports(logger)
	baseServer := base.NewBase(serverTransport)
	baseClient := base.NewBase(clientTransport)

	toolsClient := NewClient(baseClient)

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

	return ctx, toolsClient, baseServer, cleanup
}

func TestClient_List(t *testing.T) {
	tests := []struct {
		name    string
		tools   []types.Tool
		wantErr bool
		errCode int
		errMsg  string
	}{
		{
			name: "successful tool listing",
			tools: []types.Tool{
				{
					Name:        "get_weather",
					Description: "Get current weather information",
					InputSchema: struct {
						Type       string                 `json:"type"`
						Properties map[string]interface{} `json:"properties,omitempty"`
						Required   []string               `json:"required,omitempty"`
					}{
						Type: "object",
						Properties: map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "City name or zip code",
							},
						},
						Required: []string{"location"},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "server error",
			wantErr: true,
			errCode: types.InternalError,
			errMsg:  "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, client, server, cleanup := setupTest(t)
			defer cleanup()

			// Register request handler
			server.RegisterRequestHandler(methods.ListTools, func(ctx context.Context, params *json.RawMessage) (interface{}, error) {
				if tt.wantErr {
					return nil, types.NewError(tt.errCode, tt.errMsg)
				}
				return &types.ListToolsResult{
					Tools: tt.tools,
				}, nil
			})

			tools, err := client.List(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(tools) != len(tt.tools) {
					t.Errorf("Expected %d tools, got %d", len(tt.tools), len(tools))
					return
				}

				for i, want := range tt.tools {
					if tools[i].Name != want.Name {
						t.Errorf("Tool %d Name mismatch: got %s, want %s", i, tools[i].Name, want.Name)
					}
					if tools[i].Description != want.Description {
						t.Errorf("Tool %d Description mismatch: got %s, want %s", i, tools[i].Description, want.Description)
					}
					// Could add more detailed comparison of InputSchema if needed
				}
			}
		})
	}
}

func TestClient_Call(t *testing.T) {
	tests := []struct {
		name      string
		toolName  string
		args      map[string]interface{}
		want      *types.CallToolResult
		wantErr   bool
		errorCode int
		errorMsg  string
	}{
		{
			name:     "successful tool call",
			toolName: "get_weather",
			args: map[string]interface{}{
				"location": "New York",
			},
			want: &types.CallToolResult{
				Content: []interface{}{
					types.TextContent{
						Type: "text",
						Text: "Current weather in New York: 72°F, Partly cloudy",
					},
				},
				IsError: false,
			},
			wantErr: false,
		},
		{
			name:      "tool not found",
			toolName:  "nonexistent_tool",
			args:      map[string]interface{}{},
			wantErr:   true,
			errorCode: types.MethodNotFound,
			errorMsg:  "tool not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, client, server, cleanup := setupTest(t)
			defer cleanup()

			server.RegisterRequestHandler(methods.CallTool, func(ctx context.Context, params *json.RawMessage) (interface{}, error) {
				if tt.wantErr {
					return nil, types.NewError(tt.errorCode, tt.errorMsg)
				}
				return tt.want, nil
			})

			result, err := client.Call(ctx, tt.toolName, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Call() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if mcpErr, ok := err.(*types.ErrorResponse); ok {
					if mcpErr.Code != tt.errorCode {
						t.Errorf("Expected error code %d, got %d", tt.errorCode, mcpErr.Code)
					}
					if mcpErr.Message != tt.errorMsg {
						t.Errorf("Expected error message %s, got %s", tt.errorMsg, mcpErr.Message)
					}
				} else {
					t.Errorf("Expected MCP error, got %T", err)
				}
			} else {
				if len(result.Content) != len(tt.want.Content) {
					t.Errorf("Expected %d content items, got %d", len(tt.want.Content), len(result.Content))
				}
				if result.IsError != tt.want.IsError {
					t.Errorf("IsError mismatch: got %v, want %v", result.IsError, tt.want.IsError)
				}
			}
		})
	}
}

func TestClient_OnToolListChanged(t *testing.T) {
	ctx, client, server, cleanup := setupTest(t)
	defer cleanup()

	// Channel to track callback invocation
	callbackInvoked := make(chan struct{})

	// Register callback
	client.OnToolListChanged(func() {
		close(callbackInvoked)
	})

	// Send notification with empty struct as params
	notification := struct{}{}
	if err := server.SendNotification(ctx, methods.ToolsChanged, notification); err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	// Wait for callback with timeout
	select {
	case <-callbackInvoked:
		// Success
	case <-time.After(time.Second):
		t.Error("Timeout waiting for callback")
	}
}
