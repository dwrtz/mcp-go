package mcp_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dwrtz/mcp-go/internal/mock"
	"github.com/dwrtz/mcp-go/internal/testutil"
	"github.com/dwrtz/mcp-go/pkg/mcp"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// setupClientServer creates a client and server with all features enabled,
// starts them, performs client initialization, and returns them along with
// a cleanup function. By default, the client is configured with roots and
// sampling support, and the server is configured with resources, prompts,
// and tools. Both can be customized if needed.
func setupClientServer(t *testing.T) (*mcp.Client, *mcp.Server, context.Context, func()) {
	t.Helper()

	logger := testutil.NewTestLogger(t)

	// Create in-process (pipe-based) mock transports
	serverTransport, clientTransport := mock.NewMockPipeTransports(logger)

	// Create a server with resources, prompts, and tools enabled
	server := mcp.NewServer(
		serverTransport,
		mcp.WithResources(
			[]types.Resource{
				{
					URI:      "file:///example.txt",
					Name:     "Example File",
					MimeType: "text/plain",
				},
			},
			[]types.ResourceTemplate{
				{
					URITemplate: "file:///example/{name}.txt",
					Name:        "Example Text Template",
					MimeType:    "text/plain",
				},
			},
		),
		mcp.WithPrompts(
			[]types.Prompt{
				{
					Name:        "example_prompt",
					Description: "An example prompt",
					Arguments: []types.PromptArgument{
						{Name: "arg1", Required: true},
					},
				},
			},
		),
		mcp.WithTools(
			[]types.Tool{
				{
					Name:        "echo_tool",
					Description: "Echoes back the provided input",
					InputSchema: struct {
						Type       string                 `json:"type"`
						Properties map[string]interface{} `json:"properties,omitempty"`
						Required   []string               `json:"required,omitempty"`
					}{
						Type: "object",
						Properties: map[string]interface{}{
							"value": map[string]interface{}{
								"type":        "string",
								"description": "Value to echo back",
							},
						},
						Required: []string{"value"},
					},
				},
			},
		),
	)

	// Register a content handler on the server for reading resources
	server.RegisterContentHandler("file://", func(ctx context.Context, uri string) ([]types.ResourceContent, error) {
		// For demonstration, if the file name is "example.txt", return some text
		if uri == "file:///example.txt" {
			return []types.ResourceContent{
				types.TextResourceContents{
					ResourceContents: types.ResourceContents{
						URI:      "file:///example.txt",
						MimeType: "text/plain",
					},
					Text: "This is an example file content.",
				},
			}, nil
		}
		// Otherwise, return an error
		return nil, types.NewError(types.InvalidParams, "resource not found")
	})

	// Register a prompt getter on the server
	server.RegisterPromptGetter("example_prompt", func(ctx context.Context, args map[string]string) (*types.GetPromptResult, error) {
		arg1 := args["arg1"]
		result := &types.GetPromptResult{
			Description: "An example prompt result",
			Messages: []types.PromptMessage{
				{
					Role: types.RoleUser,
					Content: types.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Prompt with arg1=%s", arg1),
					},
				},
			},
		}
		return result, nil
	})

	// Register a tool handler on the server
	server.RegisterToolHandler("echo_tool", func(ctx context.Context, arguments map[string]interface{}) (*types.CallToolResult, error) {
		val, ok := arguments["value"].(string)
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
		return &types.CallToolResult{
			Content: []interface{}{
				types.TextContent{
					Type: "text",
					Text: "Echo: " + val,
				},
			},
			IsError: false,
		}, nil
	})

	// Create a client with roots and sampling support
	client := mcp.NewClient(
		clientTransport,
		mcp.WithRoots([]types.Root{
			{
				URI:  "file:///initialRoot",
				Name: "Initial Root",
			},
		}),
		mcp.WithSampling(func(ctx context.Context, req *types.CreateMessageRequest) (*types.CreateMessageResult, error) {
			// Basic sampling handler mock
			if len(req.Messages) == 0 {
				return nil, types.NewError(types.InvalidParams, "messages array cannot be empty")
			}
			if req.MaxTokens <= 0 {
				return nil, types.NewError(types.InvalidParams, "maxTokens must be positive")
			}
			return &types.CreateMessageResult{
				Role: types.RoleAssistant,
				Content: types.TextContent{
					Type: "text",
					Text: "Sampled response",
				},
				Model:      "mock-model",
				StopReason: "endTurn",
			}, nil
		}),
	)

	ctx := context.Background()

	// Start server and client
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	if err := client.Start(ctx); err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}

	// Initialize the client with the server
	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("Client initialization failed: %v", err)
	}

	// Return both plus a cleanup function
	cleanup := func() {
		client.Close()
		server.Close()
	}
	return client, server, ctx, cleanup
}

func TestClientServerIntegration(t *testing.T) {
	client, server, ctx, cleanup := setupClientServer(t)
	defer cleanup()

	t.Run("TestRoots", func(t *testing.T) {
		// Verify the server can call ListRoots on the client
		rootsList, err := server.ListRoots(ctx)
		if err != nil {
			t.Fatalf("Server.ListRoots() error: %v", err)
		}
		if len(rootsList) != 1 {
			t.Fatalf("Expected 1 root from client, got %d", len(rootsList))
		}
		if rootsList[0].URI != "file:///initialRoot" {
			t.Errorf("Unexpected root URI: %v", rootsList[0].URI)
		}

		// Verify the client can set new roots
		newRoots := []types.Root{
			{
				URI:  "file:///newRoot1",
				Name: "New Root 1",
			},
			{
				URI:  "file:///newRoot2",
				Name: "New Root 2",
			},
		}
		if err := client.SetRoots(ctx, newRoots); err != nil {
			t.Fatalf("Client.SetRoots() error: %v", err)
		}
	})

	t.Run("TestResources", func(t *testing.T) {
		// Client listing resources
		res, err := client.ListResources(ctx)
		if err != nil {
			t.Fatalf("ListResources() error: %v", err)
		}
		if len(res) != 1 || res[0].URI != "file:///example.txt" {
			t.Errorf("Expected to find resource 'file:///example.txt', got %+v", res)
		}

		// Client reading resource
		contents, err := client.ReadResource(ctx, "file:///example.txt")
		if err != nil {
			t.Fatalf("ReadResource() error: %v", err)
		}
		if len(contents) != 1 {
			t.Fatalf("Expected 1 content item, got %d", len(contents))
		}
		textContent, ok := contents[0].(types.TextResourceContents)
		if !ok {
			t.Fatalf("Expected TextResourceContents, got %T", contents[0])
		}
		if textContent.Text != "This is an example file content." {
			t.Errorf("Unexpected file content: %s", textContent.Text)
		}

		// Subscribing to resource
		err = client.SubscribeResource(ctx, "file:///example.txt")
		if err != nil {
			t.Errorf("SubscribeResource() error: %v", err)
		}

		// Use a channel to detect the resource-updated notification
		updatedCh := make(chan string)
		client.OnResourceUpdated(func(uri string) {
			updatedCh <- uri
		})

		// Trigger an update on the server
		if err := server.NotifyResourceUpdated(ctx, "file:///example.txt"); err != nil {
			t.Errorf("NotifyResourceUpdated() error: %v", err)
		}

		select {
		case updatedURI := <-updatedCh:
			if updatedURI != "file:///example.txt" {
				t.Errorf("Unexpected updated URI: %s", updatedURI)
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for resource update notification")
		}

		// Unsubscribe
		err = client.UnsubscribeResource(ctx, "file:///example.txt")
		if err != nil {
			t.Errorf("UnsubscribeResource() error: %v", err)
		}
	})

	t.Run("TestPrompts", func(t *testing.T) {
		// Client listing prompts
		promptsList, err := client.ListPrompts(ctx)
		if err != nil {
			t.Fatalf("ListPrompts() error: %v", err)
		}
		if len(promptsList) != 1 || promptsList[0].Name != "example_prompt" {
			t.Errorf("Expected to find 'example_prompt', got %+v", promptsList)
		}

		// Client getting a prompt
		promptRes, err := client.GetPrompt(ctx, "example_prompt", map[string]string{"arg1": "hello"})
		if err != nil {
			t.Fatalf("GetPrompt() error: %v", err)
		}
		if len(promptRes.Messages) != 1 {
			t.Fatalf("Expected 1 message in prompt result, got %d", len(promptRes.Messages))
		}
		msg := promptRes.Messages[0]
		txt, ok := msg.Content.(types.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent, got %T", msg.Content)
		}
		expectedText := "Prompt with arg1=hello"
		if txt.Text != expectedText {
			t.Errorf("Expected text: %q, got %q", expectedText, txt.Text)
		}

		// Use channel to detect prompt list change
		promptChangedCh := make(chan struct{})
		client.OnPromptListChanged(func() {
			close(promptChangedCh)
		})

		// Server updating prompts
		err = server.SetPrompts(ctx, []types.Prompt{
			{
				Name:        "new_prompt",
				Description: "A brand-new prompt",
			},
		})
		if err != nil {
			t.Fatalf("Server.SetPrompts() error: %v", err)
		}

		// Wait for the notification
		select {
		case <-promptChangedCh:
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for prompt list changed notification")
		}
	})

	t.Run("TestTools", func(t *testing.T) {
		// Client listing tools
		toolsList, err := client.ListTools(ctx)
		if err != nil {
			t.Fatalf("ListTools() error: %v", err)
		}
		if len(toolsList) != 1 || toolsList[0].Name != "echo_tool" {
			t.Errorf("Expected 'echo_tool', got %+v", toolsList)
		}

		// Client calling a tool
		callRes, err := client.CallTool(ctx, "echo_tool", map[string]interface{}{"value": "Hello from client"})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if callRes.IsError {
			t.Errorf("Expected no error, but got an error result")
		}
		if len(callRes.Content) != 1 {
			t.Fatalf("Expected 1 content item, got %d", len(callRes.Content))
		}
		contentMap, ok := callRes.Content[0].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map[string]interface{}, got %T", callRes.Content[0])
		}
		if contentMap["text"] != "Echo: Hello from client" {
			t.Errorf("Unexpected tool response: %v", contentMap["text"])
		}

		// Use channel to detect tool list changed notification
		toolChangedCh := make(chan struct{})
		client.OnToolListChanged(func() {
			close(toolChangedCh)
		})

		// Server updating tools
		err = server.SetTools(ctx, []types.Tool{
			{
				Name:        "new_tool",
				Description: "A new tool",
				InputSchema: struct {
					Type       string                 `json:"type"`
					Properties map[string]interface{} `json:"properties,omitempty"`
					Required   []string               `json:"required,omitempty"`
				}{
					Type: "object",
				},
			},
		})
		if err != nil {
			t.Fatalf("Server.SetTools() error: %v", err)
		}

		select {
		case <-toolChangedCh:
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for tools changed notification")
		}
	})

	t.Run("TestSampling", func(t *testing.T) {
		// Server requests a sample from the client's LLM
		req := &types.CreateMessageRequest{
			Method: "sampling/createMessage",
			Messages: []types.SamplingMessage{
				{
					Role: types.RoleUser,
					Content: types.TextContent{
						Type: "text",
						Text: "Hello!",
					},
				},
			},
			MaxTokens: 50,
		}
		result, err := server.CreateMessage(ctx, req)
		if err != nil {
			t.Fatalf("Server.CreateMessage() error: %v", err)
		}
		if result.Role != types.RoleAssistant {
			t.Errorf("Expected role 'assistant', got %s", result.Role)
		}
		txt, ok := result.Content.(types.TextContent)
		if !ok {
			t.Fatalf("Expected text content, got %T", result.Content)
		}
		if txt.Text != "Sampled response" {
			t.Errorf("Unexpected sampling response: %s", txt.Text)
		}
		if result.Model != "mock-model" {
			t.Errorf("Expected model='mock-model', got %s", result.Model)
		}
	})

	t.Run("TestShutdown", func(t *testing.T) {
		// Just confirm no errors on close in subtest
		// The `defer cleanup()` will handle it.
	})
}

// TestConcurrentUsage demonstrates multiple concurrent calls to the server & client
func TestConcurrentUsage(t *testing.T) {
	client, _, ctx, cleanup := setupClientServer(t)
	defer cleanup()

	var wg sync.WaitGroup
	const concurrentCalls = 5

	// Perform concurrent resource read calls from the client
	wg.Add(concurrentCalls)
	for i := 0; i < concurrentCalls; i++ {
		go func() {
			defer wg.Done()
			_, err := client.ReadResource(ctx, "file:///example.txt")
			if err != nil {
				t.Errorf("ReadResource error: %v", err)
			}
		}()
	}

	// Perform concurrent tool calls from the client
	wg.Add(concurrentCalls)
	for i := 0; i < concurrentCalls; i++ {
		go func(idx int) {
			defer wg.Done()
			_, err := client.CallTool(ctx, "echo_tool", map[string]interface{}{
				"value": fmt.Sprintf("Hello %d", idx),
			})
			if err != nil {
				t.Errorf("CallTool error: %v", err)
			}
		}(i)
	}

	wg.Wait()
}
