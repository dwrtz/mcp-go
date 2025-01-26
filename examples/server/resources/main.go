package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dwrtz/mcp-go/pkg/mcp/logger"
	"github.com/dwrtz/mcp-go/pkg/mcp/server"
	"github.com/dwrtz/mcp-go/pkg/types"
)

func main() {
	// Create a server that has only "Resources" enabled
	s := server.NewDefaultServer(
		server.WithLogger(logger.NewStderrLogger("RES-SERVER")),
		server.WithResources(
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
	s.RegisterContentHandler("file://", func(ctx context.Context, uri string) ([]types.ResourceContent, error) {
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
	if err := s.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server start error: %v\n", err)
		os.Exit(1)
	}

	// The server is now running. Wait indefinitely.
	select {}
}
