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

// Test input type for echo tool
type EchoInput struct {
	Value string `json:"value" jsonschema:"description=Value to echo back,required"`
}

func setupTest(t *testing.T) (context.Context, *ToolsServer, *base.Base, func()) {
	logger := testutil.NewTestLogger(t)
	serverTransport, clientTransport := mock.NewMockPipeTransports(logger)
	baseServer := base.NewBase(serverTransport)
	baseClient := base.NewBase(clientTransport)

	// Create initial tools using the new TypedTool constructor
	echoTool := types.NewTool[EchoInput](
		"test_tool",
		"A test tool",
		func(ctx context.Context, input EchoInput) (*types.CallToolResult, error) {
			return &types.CallToolResult{
				Content: []interface{}{
					types.TextContent{
						Type: "text",
						Text: "Echo: " + input.Value,
					},
				},
				IsError: false,
			}, nil
		},
	)

	initialTools := []types.McpTool{echoTool}
	toolsServer := NewToolsServer(baseServer, initialTools)

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

	return ctx, toolsServer, baseClient, cleanup
}

func TestToolsServer_SetTools(t *testing.T) {
	ctx, toolsServer, client, cleanup := setupTest(t)
	defer cleanup()

	// Channel to track notification
	notificationReceived := make(chan struct{})

	// Register notification handler
	client.RegisterNotificationHandler(methods.ToolsChanged, func(ctx context.Context, params json.RawMessage) {
		close(notificationReceived)
	})

	// Create new tools
	weatherTool := types.NewTool[struct {
		Location string `json:"location" jsonschema:"description=City name or zip code,required"`
	}](
		"get_weather",
		"Fetch current weather information",
		func(ctx context.Context, input struct {
			Location string `json:"location" jsonschema:"description=City name or zip code,required"`
		}) (*types.CallToolResult, error) {
			return &types.CallToolResult{
				Content: []interface{}{
					types.TextContent{
						Type: "text",
						Text: "Weather for " + input.Location + ": Sunny",
					},
				},
			}, nil
		},
	)

	tools := []types.McpTool{weatherTool}

	if err := toolsServer.SetTools(ctx, tools); err != nil {
		t.Fatalf("Failed to set tools: %v", err)
	}

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

	// Define new tools
	tools := []types.McpTool{
		types.NewTool[struct{ Value string }](
			"tool_one",
			"The first tool",
			func(ctx context.Context, input struct{ Value string }) (*types.CallToolResult, error) {
				return &types.CallToolResult{}, nil
			},
		),
		types.NewTool[struct{ Value string }](
			"tool_two",
			"The second tool",
			func(ctx context.Context, input struct{ Value string }) (*types.CallToolResult, error) {
				return &types.CallToolResult{}, nil
			},
		),
	}

	if err := toolsServer.SetTools(ctx, tools); err != nil {
		t.Fatalf("Failed to set tools: %v", err)
	}

	// Send list request
	resp, err := client.SendRequest(ctx, methods.ListTools, &types.ListToolsRequest{
		Method: methods.ListTools,
	})
	if err != nil {
		t.Fatalf("ListTools request failed: %v", err)
	}

	var result types.ListToolsResult
	if err := json.Unmarshal(*resp.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if len(result.Tools) != len(tools) {
		t.Errorf("Expected %d tools, got %d", len(tools), len(result.Tools))
	}

	for i, tool := range tools {
		if result.Tools[i].Name != tool.GetName() {
			t.Errorf("Tool %d name mismatch: got %s, want %s", i, result.Tools[i].Name, tool.GetName())
		}
		if result.Tools[i].Description != tool.GetDescription() {
			t.Errorf("Tool %d description mismatch: got %s, want %s", i, result.Tools[i].Description, tool.GetDescription())
		}
	}
}

func TestToolsServer_CallTool(t *testing.T) {
	ctx, toolsServer, client, cleanup := setupTest(t)
	defer cleanup()

	// Set up echo tool
	echoTool := types.NewTool[EchoInput](
		"my_special_tool",
		"Do something special",
		func(ctx context.Context, input EchoInput) (*types.CallToolResult, error) {
			return &types.CallToolResult{
				Content: []interface{}{
					types.TextContent{
						Type: "text",
						Text: "Echo: " + input.Value,
					},
				},
				IsError: false,
			}, nil
		},
	)

	if err := toolsServer.SetTools(ctx, []types.McpTool{echoTool}); err != nil {
		t.Fatalf("Failed to set tools: %v", err)
	}

	// Call the tool
	callReq := &types.CallToolRequest{
		Method:    methods.CallTool,
		Name:      "my_special_tool",
		Arguments: map[string]interface{}{"value": "Hello!"},
	}
	callResp, err := client.SendRequest(ctx, methods.CallTool, callReq)
	if err != nil {
		t.Fatalf("Failed to call tool: %v", err)
	}

	var callResult types.CallToolResult
	if err := json.Unmarshal(*callResp.Result, &callResult); err != nil {
		t.Fatalf("Failed to unmarshal call result: %v", err)
	}

	if callResult.IsError {
		t.Error("Expected IsError=false, got true")
	}
	if len(callResult.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(callResult.Content))
	}

	content := callResult.Content[0].(map[string]interface{})
	if content["type"] != "text" {
		t.Errorf("Expected content type 'text', got '%v'", content["type"])
	}
	if content["text"] != "Echo: Hello!" {
		t.Errorf("Expected text 'Echo: Hello!', got '%v'", content["text"])
	}
}

func TestToolsServer_CallTool_NotFound(t *testing.T) {
	ctx, _, client, cleanup := setupTest(t)
	defer cleanup()

	callReq := &types.CallToolRequest{
		Method:    methods.CallTool,
		Name:      "unknown_tool",
		Arguments: map[string]interface{}{},
	}
	_, err := client.SendRequest(ctx, methods.CallTool, callReq)
	if err == nil {
		t.Fatal("Expected error when calling an unregistered tool, got nil")
	}

	mcpErr, ok := err.(*types.ErrorResponse)
	if !ok {
		t.Fatalf("Expected *types.ErrorResponse, got %T", err)
	}
	if mcpErr.Message != "no handler found for tool: unknown_tool" {
		t.Errorf("Unexpected error message: %v", mcpErr.Message)
	}
}
