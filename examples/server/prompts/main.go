package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dwrtz/mcp-go/pkg/logger"
	"github.com/dwrtz/mcp-go/pkg/mcp/server"
	"github.com/dwrtz/mcp-go/pkg/types"
)

func main() {

	// Create a server that has only "Prompts" enabled
	s := server.NewDefaultServer(
		server.WithLogger(logger.NewStderrLogger("PROMPTS-SERVER")),
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

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server start error: %v\n", err)
		os.Exit(1)
	}

	select {}
}
