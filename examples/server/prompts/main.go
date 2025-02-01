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

func main() {
	lg := logger.NewStderrLogger("PROMPTS-SERVER")

	// Create a server that has only "Prompts" enabled
	s := server.NewDefaultServer(
		server.WithLogger(lg),
		server.WithPrompts([]types.Prompt{
			{
				Name:        "example_prompt",
				Description: "A basic example prompt",
				Arguments: []types.PromptArgument{
					{
						Name:        "arg1",
						Description: "An example argument",
						Required:    true,
					},
				},
			},
		}),
	)

	// Register a prompt getter
	s.RegisterPromptGetter("example_prompt", func(ctx context.Context, args map[string]string) (*types.GetPromptResult, error) {
		arg1 := args["arg1"]
		return &types.GetPromptResult{
			Description: "Server-provided example prompt",
			Messages: []types.PromptMessage{
				{
					Role: types.RoleUser,
					Content: types.TextContent{
						Type: "text",
						Text: "Hello, arg1=" + arg1,
					},
				},
			},
		}, nil
	})

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
