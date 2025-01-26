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
	// We'll create a StdioTransport that reads from os.Stdin, writes to os.Stdout
	transport := stdio.NewStdioTransport(
		os.Stdin,
		os.Stdout,
		newStdioLogger("RES-SERVER"),
	)

	// Create a server that has only "Resources" enabled
	server := mcp.NewServer(
		transport,
		mcp.WithResources(
			[]types.Resource{
				{
					URI:      "file:///example.txt",
					Name:     "Example Resource",
					MimeType: "text/plain",
				},
			},
			nil, // no templates, or provide them if you wish
		),
	)

	// Optionally register a content handler for reading resources
	server.RegisterContentHandler("file://", func(ctx context.Context, uri string) ([]types.ResourceContent, error) {
		if uri == "file:///example.txt" {
			return []types.ResourceContent{
				types.TextResourceContents{
					ResourceContents: types.ResourceContents{
						URI:      uri,
						MimeType: "text/plain",
					},
					Text: "Hello from the resources-only server!",
				},
			}, nil
		}
		return nil, types.NewError(types.InvalidParams, "Resource not found: "+uri)
	})

	// Start the server
	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server start error: %v\n", err)
		os.Exit(1)
	}

	// The server is now running. Wait indefinitely.
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
