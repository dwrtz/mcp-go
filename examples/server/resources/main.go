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
	lg := logger.NewStderrLogger("RES-SERVER")

	// Create a server that has only "Resources" enabled
	s := server.NewDefaultServer(
		server.WithLogger(lg),
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
