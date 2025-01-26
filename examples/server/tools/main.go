package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dwrtz/mcp-go/internal/transport/stdio"
	"github.com/dwrtz/mcp-go/pkg/mcp"
	"github.com/dwrtz/mcp-go/pkg/types"
)

func main() {
	transport := stdio.NewStdioTransport(
		os.Stdin,
		os.Stdout,
		newStdioLogger("TOOLS-SERVER"),
	)

	// Create a server that has only "Tools" enabled
	server := mcp.NewServer(
		transport,
		mcp.WithTools([]types.Tool{
			{
				Name:        "echo_tool",
				Description: "Echoes back the input in 'value' argument",
				InputSchema: struct {
					Type       string                 `json:"type"`
					Properties map[string]interface{} `json:"properties,omitempty"`
					Required   []string               `json:"required,omitempty"`
				}{
					Type: "object",
					Properties: map[string]interface{}{
						"value": map[string]interface{}{
							"type":        "string",
							"description": "Input to echo",
						},
					},
					Required: []string{"value"},
				},
			},
		}),
	)

	// Register a tool handler
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
					Text: "[TOOLS-SERVER] Echo: " + val,
				},
			},
			IsError: false,
		}, nil
	})

	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server start error: %v\n", err)
		os.Exit(1)
	}

	select {}
}

func newStdioLogger(prefix string) *stdioLogger {
	return &stdioLogger{prefix: prefix}
}

type stdioLogger struct {
	prefix string
}

func (l *stdioLogger) Logf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[%s] ", l.prefix)
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
