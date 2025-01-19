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

// setupTest creates a server, a client, and a ToolsServer instance, then starts them.
// It returns a cleanup function that should be deferred to properly close everything.
func setupTest(t *testing.T) (context.Context, *ToolsServer, *base.Base, func()) {
	logger := testutil.NewTestLogger(t)
	serverTransport, clientTransport := mock.NewMockPipeTransports(logger)

	// Create base server and client
	baseServer := base.NewBase(serverTransport)
	baseClient := base.NewBase(clientTransport)

	// Create tools server
	toolsServer := NewToolsServer(baseServer)

	// Start both
	ctx := context.Background()
	if err := baseServer.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	if err := baseClient.Start(ctx); err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}

	// Provide cleanup function
	cleanup := func() {
		baseClient.Close()
		baseServer.Close()
	}

	return ctx, toolsServer, baseClient, cleanup
}

func TestToolsServer_SetTools(t *testing.T) {
	ctx, toolsServer, client, cleanup := setupTest(t)
	defer cleanup()

	// Channel to track notification
	notificationReceived := make(chan struct{})

	// Register notification handler on the client side
	client.RegisterNotificationHandler(methods.ToolsChanged, func(ctx context.Context, params json.RawMessage) {
		close(notificationReceived)
	})

	// Prepare some tools
	tools := []types.Tool{
		{
			Name:        "get_weather",
			Description: "Fetch current weather information",
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
		{
			Name:        "send_email",
			Description: "Send an email message",
			InputSchema: struct {
				Type       string                 `json:"type"`
				Properties map[string]interface{} `json:"properties,omitempty"`
				Required   []string               `json:"required,omitempty"`
			}{
				Type: "object",
				Properties: map[string]interface{}{
					"to":   map[string]interface{}{"type": "string"},
					"body": map[string]interface{}{"type": "string"},
				},
				Required: []string{"to", "body"},
			},
		},
	}

	// Call SetTools, which should trigger a ToolsChanged notification
	if err := toolsServer.SetTools(ctx, tools); err != nil {
		t.Fatalf("Failed to set tools: %v", err)
	}

	// Wait for notification with timeout
	select {
	case <-notificationReceived:
		// Success
	case <-time.After(time.Second):
		t.Error("Timeout waiting for ToolsChanged notification")
	}
}

func TestToolsServer_ListTools(t *testing.T) {
	ctx, toolsServer, client, cleanup := setupTest(t)
	defer cleanup()

	// Define initial tools
	tools := []types.Tool{
		{
			Name:        "tool_one",
			Description: "The first tool",
			InputSchema: struct {
				Type       string                 `json:"type"`
				Properties map[string]interface{} `json:"properties,omitempty"`
				Required   []string               `json:"required,omitempty"`
			}{
				Type: "object",
			},
		},
		{
			Name:        "tool_two",
			Description: "The second tool",
			InputSchema: struct {
				Type       string                 `json:"type"`
				Properties map[string]interface{} `json:"properties,omitempty"`
				Required   []string               `json:"required,omitempty"`
			}{
				Type: "object",
			},
		},
	}

	// Set tools on server
	if err := toolsServer.SetTools(ctx, tools); err != nil {
		t.Fatalf("Failed to set tools: %v", err)
	}

	// Send a list-tools request from the client
	req := &types.ListToolsRequest{
		Method: methods.ListTools,
	}
	resp, err := client.SendRequest(ctx, methods.ListTools, req)
	if err != nil {
		t.Fatalf("ListTools request failed: %v", err)
	}

	// Parse response
	var result types.ListToolsResult
	if err := json.Unmarshal(*resp.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Verify the listed tools match what was set
	if len(result.Tools) != len(tools) {
		t.Errorf("Expected %d tools, got %d", len(tools), len(result.Tools))
	}

	for i, want := range tools {
		if i >= len(result.Tools) {
			t.Errorf("Missing tool at index %d", i)
			continue
		}
		got := result.Tools[i]
		if got.Name != want.Name {
			t.Errorf("Tool %d name mismatch: got %s, want %s", i, got.Name, want.Name)
		}
		if got.Description != want.Description {
			t.Errorf("Tool %d description mismatch: got %s, want %s", i, got.Description, want.Description)
		}
	}
}

func TestToolsServer_CallTool(t *testing.T) {
	ctx, toolsServer, client, cleanup := setupTest(t)
	defer cleanup()

	// Prepare a tool
	tool := types.Tool{
		Name:        "my_special_tool",
		Description: "Do something special",
		InputSchema: struct {
			Type       string                 `json:"type"`
			Properties map[string]interface{} `json:"properties,omitempty"`
			Required   []string               `json:"required,omitempty"`
		}{
			Type: "object",
			Properties: map[string]interface{}{
				"value": map[string]interface{}{"type": "string"},
			},
			Required: []string{"value"},
		},
	}

	// Set the tool
	if err := toolsServer.SetTools(ctx, []types.Tool{tool}); err != nil {
		t.Fatalf("Failed to set tools: %v", err)
	}

	// Register the tool handler
	toolsServer.RegisterToolHandler("my_special_tool", func(ctx context.Context, args map[string]interface{}) (*types.CallToolResult, error) {
		val, ok := args["value"].(string)
		if !ok {
			return &types.CallToolResult{
				Content: []interface{}{
					types.TextContent{
						Type: "text",
						Text: "Error: 'value' must be a string",
					},
				},
				IsError: true,
			}, nil
		}
		// Return a non-error result
		return &types.CallToolResult{
			Content: []interface{}{
				types.TextContent{
					Type: "text",
					Text: "Received value: " + val,
				},
			},
			IsError: false,
		}, nil
	})

	// Make a call to the tool
	callReq := &types.CallToolRequest{
		Method:    methods.CallTool,
		Name:      "my_special_tool",
		Arguments: map[string]interface{}{"value": "Hello!"},
	}
	callResp, err := client.SendRequest(ctx, methods.CallTool, callReq)
	if err != nil {
		t.Fatalf("Failed to call tool: %v", err)
	}

	// Parse tool call result
	var callResult types.CallToolResult
	if err := json.Unmarshal(*callResp.Result, &callResult); err != nil {
		t.Fatalf("Failed to unmarshal call result: %v", err)
	}

	// Verify
	if callResult.IsError {
		t.Error("Expected IsError=false, got true")
	}
	if len(callResult.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(callResult.Content))
	}
	txt, ok := callResult.Content[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected text content object, got %T", callResult.Content[0])
	}
	if txt["type"] != "text" {
		t.Errorf("Expected content type 'text', got '%v'", txt["type"])
	}
	if txt["text"] != "Received value: Hello!" {
		t.Errorf("Expected text 'Received value: Hello!', got '%v'", txt["text"])
	}
}

func TestToolsServer_CallTool_NotFound(t *testing.T) {
	ctx, _, client, cleanup := setupTest(t)
	defer cleanup()

	// Attempt calling a tool that hasn't been registered
	callReq := &types.CallToolRequest{
		Method:    methods.CallTool,
		Name:      "unknown_tool",
		Arguments: map[string]interface{}{},
	}
	_, err := client.SendRequest(ctx, methods.CallTool, callReq)
	if err == nil {
		t.Fatal("Expected error when calling an unregistered tool, got nil")
	}

	// Verify it's the correct type of error
	mcpErr, ok := err.(*types.ErrorResponse)
	if !ok {
		t.Fatalf("Expected *types.ErrorResponse, got %T", err)
	}
	if mcpErr.Message != "no handler found for tool: unknown_tool" {
		t.Errorf("Unexpected error message: %v", mcpErr.Message)
	}
}
