package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dwrtz/mcp-go/pkg/logger"
	"github.com/dwrtz/mcp-go/pkg/mcp/server"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// EchoInput defines the input type for the echo tool
type EchoInput struct {
	Value string `json:"value" jsonschema:"description=Input to echo,required"`
}

func main() {
	lg := logger.NewStderrLogger("TOOLS-SERVER")

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
		server.WithLogger(lg),
		server.WithTools(echoTool),
	)

	// Create a context that can be canceled when the server is stopped
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server start error: %v\n", err)
		os.Exit(1)
	}

	// Set up OS signal handling for graceful shutdown (e.g. Ctrl+C)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait until either a termination signal is received or the transport is closed.
	select {
	case sig := <-sigCh:
		fmt.Printf("Received signal %v. Shutting down...\n", sig)
	case <-s.Done():
		fmt.Println("Client disconnected. Shutting down server...")
	case <-ctx.Done():
		fmt.Println("Context canceled. Shutting down server...")
	}

	lg.Logf("Exiting...")
}
