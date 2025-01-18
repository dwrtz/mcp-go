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

func setupTestToolsClient(t *testing.T) (*ToolsClient, *mock.MockTransport, context.Context, context.CancelFunc) {
	logger := testutil.NewTestLogger(t)
	mockTransport := mock.NewMockTransport(logger)
	baseClient := base.NewClient(mockTransport)
	client := NewToolsClient(baseClient)

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
						Type       string                 "json:\"type\""
						Properties map[string]interface{} "json:\"properties,omitempty\""
						Required   []string               "json:\"required,omitempty\""
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
				{
					Name:        "calculate_distance",
					Description: "Calculate distance between two points",
					InputSchema: struct {
						Type       string                 "json:\"type\""
						Properties map[string]interface{} "json:\"properties,omitempty\""
						Required   []string               "json:\"required,omitempty\""
					}{
						Type: "object",
						Properties: map[string]interface{}{
							"point1": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"lat": map[string]interface{}{"type": "number"},
									"lng": map[string]interface{}{"type": "number"},
								},
							},
							"point2": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"lat": map[string]interface{}{"type": "number"},
									"lng": map[string]interface{}{"type": "number"},
								},
							},
						},
						Required: []string{"point1", "point2"},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "empty tool list",
			tools:   []types.Tool{},
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
			client, mockTransport, ctx, cancel := setupTestToolsClient(t)
			defer cancel()

			mockTransport.ClearSentMessages()
			done := make(chan struct{})

			go func() {
				defer close(done)
				select {
				case msg := <-mockTransport.GetRouter().Requests:
					if msg.Method != methods.ListTools {
						t.Errorf("Expected method %s, got %s", methods.ListTools, msg.Method)
					}

					response := &types.Message{
						JSONRPC: types.JSONRPCVersion,
						ID:      msg.ID,
					}

					if tt.wantErr {
						response.Error = &types.ErrorResponse{
							Code:    types.InternalError,
							Message: tt.errMsg,
						}
					} else {
						result := &types.ListToolsResult{
							Tools: tt.tools,
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

			tools, err := client.List(ctx)
			<-done

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
		{
			name:     "invalid arguments",
			toolName: "get_weather",
			args: map[string]interface{}{
				"invalid_param": "value",
			},
			wantErr:   true,
			errorCode: types.InvalidParams,
			errorMsg:  "invalid parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockTransport, ctx, cancel := setupTestToolsClient(t)
			defer cancel()

			mockTransport.ClearSentMessages()
			done := make(chan struct{})

			go func() {
				defer close(done)
				select {
				case msg := <-mockTransport.GetRouter().Requests:
					if msg.Method != methods.CallTool {
						t.Errorf("Expected method %s, got %s", methods.CallTool, msg.Method)
					}

					var req types.CallToolRequest
					if err := json.Unmarshal(*msg.Params, &req); err != nil {
						t.Errorf("Failed to unmarshal request params: %v", err)
						return
					}

					if req.Name != tt.toolName {
						t.Errorf("Expected tool name %s, got %s", tt.toolName, req.Name)
					}

					response := &types.Message{
						JSONRPC: types.JSONRPCVersion,
						ID:      msg.ID,
					}

					if tt.wantErr {
						response.Error = &types.ErrorResponse{
							Code:    tt.errorCode,
							Message: tt.errorMsg,
						}
					} else {
						data, err := testutil.MarshalResult(tt.want)
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

			result, err := client.Call(ctx, tt.toolName, tt.args)
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

				// Compare content (this is a simplified comparison)
				for i, want := range tt.want.Content {
					if i >= len(result.Content) {
						t.Errorf("Missing content at index %d", i)
						continue
					}
					wantJSON, _ := json.Marshal(want)
					gotJSON, _ := json.Marshal(result.Content[i])
					wantRaw := json.RawMessage(wantJSON)
					gotRaw := json.RawMessage(gotJSON)
					if !testutil.JSONEqual(t, &wantRaw, &gotRaw) {
						t.Errorf("Content %d mismatch:\nwant: %s\ngot:  %s", i, wantJSON, gotJSON)
					}
				}

				if result.IsError != tt.want.IsError {
					t.Errorf("IsError mismatch: want %v, got %v", tt.want.IsError, result.IsError)
				}
			}
		})
	}
}

func TestToolsClient_ToolListChanged(t *testing.T) {
	client, mockTransport, ctx, cancel := setupTestToolsClient(t)
	defer cancel()

	// Channel to track callback invocation
	callbackInvoked := make(chan struct{})

	// Register callback
	client.OnToolListChanged(func() {
		close(callbackInvoked)
	})

	// Simulate server sending a notification
	notification := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		Method:  methods.ToolsChanged,
		Params:  &json.RawMessage{'{', '}'}, // Empty JSON object
	}
	mockTransport.SimulateReceive(ctx, notification)

	// Wait for callback with timeout
	select {
	case <-callbackInvoked:
		// Success - callback was invoked
	case <-time.After(time.Second):
		t.Error("Timeout waiting for tool list changed callback")
	case <-ctx.Done():
		t.Error("Context cancelled while waiting for callback")
	}
}
