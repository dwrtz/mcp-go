package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dwrtz/mcp-go/pkg/logger"
	"github.com/dwrtz/mcp-go/pkg/mcp/server"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// EchoInput defines the input type for the echo tool
type EchoInput struct {
	Value string `json:"value" jsonschema:"description=Input to echo,required"`
}

func main() {
	// Create an echo tool using the typed NewTool constructor
	echoTool := types.NewTool(
		"echo_tool",
		"Echoes back the input in 'value' argument",
		func(ctx context.Context, input EchoInput) (*types.CallToolResult, error) {
			return &types.CallToolResult{
				Content: []interface{}{
					types.TextContent{
						Type: "text",
						Text: "[TOOLS-SERVER] Echo: " + input.Value,
					},
				},
				IsError: false,
			}, nil
		},
	)

	// Create server with tools
	s := server.NewDefaultServer(
		server.WithLogger(logger.NewStderrLogger("TOOLS-SERVER")),
		server.WithTools([]types.McpTool{echoTool}),
	)

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server start error: %v\n", err)
		os.Exit(1)
	}

	select {}
}
