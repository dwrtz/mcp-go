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
		newStdioLogger("PROMPTS-SERVER"),
	)

	// Create a server that has only "Prompts" enabled
	server := mcp.NewServer(
		transport,
		mcp.WithPrompts([]types.Prompt{
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
	server.RegisterPromptGetter("example_prompt", func(ctx context.Context, args map[string]string) (*types.GetPromptResult, error) {
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
