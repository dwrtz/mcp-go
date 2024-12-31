package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dwrtz/mcp-go/internal/client"
	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/internal/transport/stdio"
)

func main() {
	// Create client transport over stdio
	cliTransport := stdio.NewStdioTransport(&transport.Options{
		Handler: nil, // We'll assign a handler in client.Start(...)
	})

	cli := client.NewClient(cliTransport)
	ctx := context.Background()

	const pingID = "ping-1"

	// Start reading in a goroutine so we can receive the server's response
	err := cli.Start(ctx, pingID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting client transport: %v\n", err)
		os.Exit(1)
	}

	// Send a "ping" request
	err = cli.Ping(ctx, pingID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending ping request: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Client: Ping sent. Waiting for server response...")

	// Wait for either the client transport to close (on "pong") or context canceled
	select {
	case <-cli.Done():
		fmt.Fprintln(os.Stderr, "Client: Transport closed. Exiting.")
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "Client: Context canceled. Exiting.")
	}

	fmt.Fprintln(os.Stderr, "Client: main() returning now.")
}
