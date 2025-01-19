package client

import (
	"testing"

	"github.com/dwrtz/mcp-go/internal/mock"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

func TestToolsClient_List(t *testing.T) {
	tests := []struct {
		name    string
		tools   []types.Tool
		wantErr bool
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
			errMsg:  "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client with all dependencies
			mockClient := mock.NewMockClient(t)
			defer mockClient.Close()

			// Create tools client
			client := NewToolsClient(mockClient.BaseClient)

			// Handle the expected request
			done := mockClient.ExpectRequest(methods.ListTools, func(msg *types.Message) *types.Message {
				if tt.wantErr {
					return mockClient.CreateErrorResponse(msg, types.InternalError, tt.errMsg, nil)
				}
				return mockClient.CreateSuccessResponse(msg, &types.ListToolsResult{
					Tools: tt.tools,
				})
			})

			// Make the actual request
			tools, err := client.List(mockClient.Context)

			// Wait for mock handler to complete
			<-done

			// Verify results
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(tools) != len(tt.tools) {
					t.Errorf("Expected %d tools, got %d", len(tt.tools), len(tools))
				}

				for i, want := range tt.tools {
					if i >= len(tools) {
						t.Errorf("Missing tool at index %d", i)
						continue
					}
					if tools[i].Name != want.Name {
						t.Errorf("Tool %d Name mismatch: want %s, got %s", i, want.Name, tools[i].Name)
					}
					if tools[i].Description != want.Description {
						t.Errorf("Tool %d Description mismatch: want %s, got %s", i, want.Description, tools[i].Description)
					}
				}
			}

			mockClient.AssertNoErrors(t)
		})
	}
}

func TestToolsClient_Call(t *testing.T) {
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
			mockClient := mock.NewMockClient(t)
			defer mockClient.Close()

			client := NewToolsClient(mockClient.BaseClient)

			done := mockClient.ExpectRequest(methods.CallTool, func(msg *types.Message) *types.Message {
				if tt.wantErr {
					return mockClient.CreateErrorResponse(msg, tt.errorCode, tt.errorMsg, nil)
				}
				return mockClient.CreateSuccessResponse(msg, tt.want)
			})

			result, err := client.Call(mockClient.Context, tt.toolName, tt.args)
			<-done

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
					t.Errorf("IsError mismatch: want %v, got %v", tt.want.IsError, result.IsError)
				}
			}

			mockClient.AssertNoErrors(t)
		})
	}
}

func TestToolsClient_OnToolListChanged(t *testing.T) {
	mockClient := mock.NewMockClient(t)
	defer mockClient.Close()

	client := NewToolsClient(mockClient.BaseClient)

	// Channel to track callback invocation
	callbackInvoked := make(chan struct{})

	// Register callback
	client.OnToolListChanged(func() {
		close(callbackInvoked)
	})

	// Test notification handling
	err := mockClient.SimulateNotification(methods.ToolsChanged, struct{}{})
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
