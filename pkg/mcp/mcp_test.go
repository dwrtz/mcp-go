package mcp_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dwrtz/mcp-go/internal/mock"
	"github.com/dwrtz/mcp-go/internal/testutil"
	"github.com/dwrtz/mcp-go/pkg/mcp/client"
	"github.com/dwrtz/mcp-go/pkg/mcp/server"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// Input types for tools
type EchoInput struct {
	Value string `json:"value" jsonschema:"description=Value to echo back,required"`
}

func setupClientServer(t *testing.T) (*client.Client, *server.Server, context.Context, func()) {
	t.Helper()

	logger := testutil.NewTestLogger(t)
	serverTransport, clientTransport := mock.NewMockPipeTransports(logger)

	// Create echo tool with typed input/output
	echoTool := types.NewTool[EchoInput](
		"echo_tool",
		"Echoes back the provided input",
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

	// Create a server with resources, prompts, and tools enabled
	s := server.NewServer(
		serverTransport,
		server.WithResources(
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
		server.WithPrompts(
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
		server.WithTools(echoTool),
	)

	// Register content handler for resources
	s.RegisterContentHandler("file://", func(ctx context.Context, uri string) ([]types.ResourceContent, error) {
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
		return nil, types.NewError(types.InvalidParams, "resource not found")
	})

	// Register prompt getter
	s.RegisterPromptGetter("example_prompt", func(ctx context.Context, args map[string]string) (*types.GetPromptResult, error) {
		arg1 := args["arg1"]
		return &types.GetPromptResult{
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
		}, nil
	})

	// Create client with roots and sampling support
	c := client.NewClient(
		clientTransport,
		client.WithRoots([]types.Root{
			{
				URI:  "file:///initialRoot",
				Name: "Initial Root",
			},
		}),
		client.WithSampling(func(ctx context.Context, req *types.CreateMessageRequest) (*types.CreateMessageResult, error) {
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

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	if err := c.Start(ctx); err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}

	if err := c.Initialize(ctx); err != nil {
		t.Fatalf("Client initialization failed: %v", err)
	}

	cleanup := func() {
		c.Close()
		s.Close()
	}
	return c, s, ctx, cleanup
}

func TestClientServerIntegration(t *testing.T) {
	c, s, ctx, cleanup := setupClientServer(t)
	defer cleanup()

	t.Run("TestRoots", func(t *testing.T) {
		rootsList, err := s.ListRoots(ctx)
		if err != nil {
			t.Fatalf("Server.ListRoots() error: %v", err)
		}
		if len(rootsList) != 1 || rootsList[0].URI != "file:///initialRoot" {
			t.Errorf("Unexpected root: %+v", rootsList)
		}

		newRoots := []types.Root{
			{URI: "file:///newRoot1", Name: "New Root 1"},
			{URI: "file:///newRoot2", Name: "New Root 2"},
		}
		if err := c.SetRoots(ctx, newRoots); err != nil {
			t.Fatalf("Client.SetRoots() error: %v", err)
		}
	})

	t.Run("TestResources", func(t *testing.T) {
		res, err := c.ListResources(ctx)
		if err != nil {
			t.Fatalf("ListResources() error: %v", err)
		}
		if len(res) != 1 || res[0].URI != "file:///example.txt" {
			t.Errorf("Expected resource 'file:///example.txt', got %+v", res)
		}

		contents, err := c.ReadResource(ctx, "file:///example.txt")
		if err != nil {
			t.Fatalf("ReadResource() error: %v", err)
		}
		textContent, ok := contents[0].(types.TextResourceContents)
		if !ok || textContent.Text != "This is an example file content." {
			t.Errorf("Unexpected content: %+v", contents[0])
		}

		updatedCh := make(chan string)
		c.OnResourceUpdated(func(uri string) {
			updatedCh <- uri
		})

		if err := c.SubscribeResource(ctx, "file:///example.txt"); err != nil {
			t.Fatalf("SubscribeResource() error: %v", err)
		}

		if err := s.NotifyResourceUpdated(ctx, "file:///example.txt"); err != nil {
			t.Fatalf("NotifyResourceUpdated() error: %v", err)
		}

		select {
		case uri := <-updatedCh:
			if uri != "file:///example.txt" {
				t.Errorf("Unexpected URI: %s", uri)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for update")
		}

		if err := c.UnsubscribeResource(ctx, "file:///example.txt"); err != nil {
			t.Fatalf("UnsubscribeResource() error: %v", err)
		}
	})

	t.Run("TestPrompts", func(t *testing.T) {
		prompts, err := c.ListPrompts(ctx)
		if err != nil {
			t.Fatalf("ListPrompts() error: %v", err)
		}
		if len(prompts) != 1 || prompts[0].Name != "example_prompt" {
			t.Errorf("Unexpected prompts: %+v", prompts)
		}

		result, err := c.GetPrompt(ctx, "example_prompt", map[string]string{"arg1": "test"})
		if err != nil {
			t.Fatalf("GetPrompt() error: %v", err)
		}
		if txt, ok := result.Messages[0].Content.(types.TextContent); !ok || txt.Text != "Prompt with arg1=test" {
			t.Errorf("Unexpected prompt content: %+v", result.Messages[0].Content)
		}
	})

	t.Run("TestTools", func(t *testing.T) {
		tools, err := c.ListTools(ctx)
		if err != nil {
			t.Fatalf("ListTools() error: %v", err)
		}
		if len(tools) != 1 || tools[0].Name != "echo_tool" {
			t.Errorf("Unexpected tools: %+v", tools)
		}

		result, err := c.CallTool(ctx, "echo_tool", map[string]interface{}{
			"value": "test message",
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		content := result.Content[0].(map[string]interface{})
		if content["text"] != "Echo: test message" {
			t.Errorf("Unexpected tool response: %v", content["text"])
		}
	})

	t.Run("TestSampling", func(t *testing.T) {
		req := &types.CreateMessageRequest{
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
		result, err := s.CreateMessage(ctx, req)
		if err != nil {
			t.Fatalf("CreateMessage() error: %v", err)
		}
		if result.Role != types.RoleAssistant {
			t.Errorf("Expected role 'assistant', got %s", result.Role)
		}
		if txt, ok := result.Content.(types.TextContent); !ok || txt.Text != "Sampled response" {
			t.Errorf("Unexpected sampling response: %+v", result.Content)
		}
	})
}

func TestConcurrentUsage(t *testing.T) {
	c, _, ctx, cleanup := setupClientServer(t)
	defer cleanup()

	var wg sync.WaitGroup
	const concurrentCalls = 5

	wg.Add(concurrentCalls)
	for i := 0; i < concurrentCalls; i++ {
		go func() {
			defer wg.Done()
			if _, err := c.ReadResource(ctx, "file:///example.txt"); err != nil {
				t.Errorf("ReadResource error: %v", err)
			}
		}()
	}

	wg.Add(concurrentCalls)
	for i := 0; i < concurrentCalls; i++ {
		go func(idx int) {
			defer wg.Done()
			if _, err := c.CallTool(ctx, "echo_tool", map[string]interface{}{
				"value": fmt.Sprintf("Hello %d", idx),
			}); err != nil {
				t.Errorf("CallTool error: %v", err)
			}
		}(i)
	}

	wg.Wait()
}

// setupSseClientServer starts an SSE-based MCP server on the hardcoded port 42069
// and connects an SSE-based MCP client. It returns the client, server, context,
// and a cleanup function.
func setupSseClientServer(t *testing.T) (*client.Client, *server.Server, context.Context, func()) {
	t.Helper()

	//------------------------------------------------------------
	// 1. Create an SSE server using NewSseServer(addr, ...options...)
	//------------------------------------------------------------
	logger := testutil.NewTestLogger(t)

	// Create an echo tool with typed input/output
	echoTool := types.NewTool[EchoInput](
		"echo_tool",
		"Echoes back the provided input",
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

	s := server.NewSseServer(
		"127.0.0.1:42069",
		server.WithLogger(logger),
		server.WithResources(
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
		server.WithPrompts(
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
		server.WithTools(echoTool),
	)

	// Register content handler for resources
	s.RegisterContentHandler("file://", func(ctx context.Context, uri string) ([]types.ResourceContent, error) {
		if uri == "file:///example.txt" {
			return []types.ResourceContent{
				types.TextResourceContents{
					ResourceContents: types.ResourceContents{
						URI:      uri,
						MimeType: "text/plain",
					},
					Text: "This is an example file content.",
				},
			}, nil
		}
		return nil, types.NewError(types.InvalidParams, "resource not found")
	})

	// Register prompt getter
	s.RegisterPromptGetter("example_prompt", func(ctx context.Context, args map[string]string) (*types.GetPromptResult, error) {
		arg1 := args["arg1"]
		return &types.GetPromptResult{
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
		}, nil
	})

	//------------------------------------------------------------
	// 2. Create an SSE client that connects to the same address
	//------------------------------------------------------------
	sseClient, err := client.NewSseClient(context.Background(), "127.0.0.1:42069",
		client.WithLogger(logger),
		client.WithRoots([]types.Root{
			{
				URI:  "file:///initialRoot",
				Name: "Initial Root",
			},
		}),
		client.WithSampling(func(ctx context.Context, req *types.CreateMessageRequest) (*types.CreateMessageResult, error) {
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
	if err != nil {
		t.Fatalf("failed to create SSE client: %v", err)
	}

	//------------------------------------------------------------
	// 3. Actually start the SSE server & client
	//------------------------------------------------------------
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Failed to start SSE server: %v", err)
	}
	if err := sseClient.Start(ctx); err != nil {
		t.Fatalf("Failed to start SSE client: %v", err)
	}

	// Perform MCP Initialize handshake
	if err := sseClient.Initialize(ctx); err != nil {
		t.Fatalf("SSE client initialization failed: %v", err)
	}

	//------------------------------------------------------------
	// 4. Cleanup function for test tear-down
	//------------------------------------------------------------
	cleanup := func() {
		sseClient.Close()
		s.Close()
	}

	// Return the SSE-based client, server, etc.
	return sseClient, s, ctx, cleanup
}

// TestSseClientServerIntegration replicates your stdio-based tests using SSE transport.
func TestSseClientServerIntegration(t *testing.T) {
	c, s, ctx, cleanup := setupSseClientServer(t)
	defer cleanup()

	t.Run("TestRoots", func(t *testing.T) {
		rootsList, err := s.ListRoots(ctx)
		if err != nil {
			t.Fatalf("Server.ListRoots() error: %v", err)
		}
		if len(rootsList) != 1 || rootsList[0].URI != "file:///initialRoot" {
			t.Errorf("Unexpected root: %+v", rootsList)
		}

		newRoots := []types.Root{
			{URI: "file:///newRoot1", Name: "New Root 1"},
			{URI: "file:///newRoot2", Name: "New Root 2"},
		}
		if err := c.SetRoots(ctx, newRoots); err != nil {
			t.Fatalf("Client.SetRoots() error: %v", err)
		}
	})

	t.Run("TestResources", func(t *testing.T) {
		res, err := c.ListResources(ctx)
		if err != nil {
			t.Fatalf("ListResources() error: %v", err)
		}
		if len(res) != 1 || res[0].URI != "file:///example.txt" {
			t.Errorf("Expected resource 'file:///example.txt', got %+v", res)
		}

		contents, err := c.ReadResource(ctx, "file:///example.txt")
		if err != nil {
			t.Fatalf("ReadResource() error: %v", err)
		}
		textContent, ok := contents[0].(types.TextResourceContents)
		if !ok || textContent.Text != "This is an example file content." {
			t.Errorf("Unexpected content: %+v", contents[0])
		}

		updatedCh := make(chan string)
		c.OnResourceUpdated(func(uri string) {
			updatedCh <- uri
		})

		if err := c.SubscribeResource(ctx, "file:///example.txt"); err != nil {
			t.Fatalf("SubscribeResource() error: %v", err)
		}

		if err := s.NotifyResourceUpdated(ctx, "file:///example.txt"); err != nil {
			t.Fatalf("NotifyResourceUpdated() error: %v", err)
		}

		select {
		case uri := <-updatedCh:
			if uri != "file:///example.txt" {
				t.Errorf("Unexpected URI: %s", uri)
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for resource update notification via SSE")
		}

		if err := c.UnsubscribeResource(ctx, "file:///example.txt"); err != nil {
			t.Fatalf("UnsubscribeResource() error: %v", err)
		}
	})

	t.Run("TestPrompts", func(t *testing.T) {
		prompts, err := c.ListPrompts(ctx)
		if err != nil {
			t.Fatalf("ListPrompts() error: %v", err)
		}
		if len(prompts) != 1 || prompts[0].Name != "example_prompt" {
			t.Errorf("Unexpected prompts: %+v", prompts)
		}

		result, err := c.GetPrompt(ctx, "example_prompt", map[string]string{"arg1": "test"})
		if err != nil {
			t.Fatalf("GetPrompt() error: %v", err)
		}
		if txt, ok := result.Messages[0].Content.(types.TextContent); !ok || txt.Text != "Prompt with arg1=test" {
			t.Errorf("Unexpected prompt content: %+v", result.Messages[0].Content)
		}
	})

	t.Run("TestTools", func(t *testing.T) {
		tools, err := c.ListTools(ctx)
		if err != nil {
			t.Fatalf("ListTools() error: %v", err)
		}
		if len(tools) != 1 || tools[0].Name != "echo_tool" {
			t.Errorf("Unexpected tools: %+v", tools)
		}

		result, err := c.CallTool(ctx, "echo_tool", map[string]interface{}{
			"value": "test message",
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		content := result.Content[0].(map[string]interface{})
		if content["text"] != "Echo: test message" {
			t.Errorf("Unexpected tool response: %v", content["text"])
		}
	})

	t.Run("TestSampling", func(t *testing.T) {
		req := &types.CreateMessageRequest{
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
		result, err := s.CreateMessage(ctx, req)
		if err != nil {
			t.Fatalf("CreateMessage() error: %v", err)
		}
		if result.Role != types.RoleAssistant {
			t.Errorf("Expected role 'assistant', got %s", result.Role)
		}
		if txt, ok := result.Content.(types.TextContent); !ok || txt.Text != "Sampled response" {
			t.Errorf("Unexpected sampling response: %+v", result.Content)
		}
	})
}

// TestSseConcurrentUsage ensures concurrency works over SSE as well
func TestSseConcurrentUsage(t *testing.T) {
	c, _, ctx, cleanup := setupSseClientServer(t)
	defer cleanup()

	var wg sync.WaitGroup
	const concurrentCalls = 5

	// Launch multiple resource reads
	wg.Add(concurrentCalls)
	for i := 0; i < concurrentCalls; i++ {
		go func() {
			defer wg.Done()
			if _, err := c.ReadResource(ctx, "file:///example.txt"); err != nil {
				t.Errorf("ReadResource error: %v", err)
			}
		}()
	}

	// Launch multiple tool calls
	wg.Add(concurrentCalls)
	for i := 0; i < concurrentCalls; i++ {
		go func(idx int) {
			defer wg.Done()
			if _, err := c.CallTool(ctx, "echo_tool", map[string]interface{}{
				"value": fmt.Sprintf("Hello %d", idx),
			}); err != nil {
				t.Errorf("CallTool error: %v", err)
			}
		}(i)
	}

	wg.Wait()
}
