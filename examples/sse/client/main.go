package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dwrtz/mcp-go/pkg/logger"
	"github.com/dwrtz/mcp-go/pkg/mcp/client"
)

func main() {
	// Command-line flag for specifying the server address
	serverAddr := flag.String("server", "localhost:8080", "Address of the SSE server (host:port)")
	flag.Parse()

	// Create and start the SSE client
	ctx := context.Background()
	c, err := client.NewSseClient(ctx, *serverAddr, client.WithLogger(logger.NewStderrLogger("SSE-CLIENT")))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Client initialization error: %v\n", err)
		os.Exit(1)
	}

	// Initialize with the server
	if err := c.Initialize(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Client initialization error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("SSE client connected & initialized with server!")

	// Demo functionality, similar to the stdio example
	if c.SupportsTools() {
		fmt.Println("Server supports Tools. Calling 'echo_tool'...")

		callRes, err := c.CallTool(ctx, "echo_tool", map[string]interface{}{
			"value": "Hello via SSE!",
		})
		if err != nil {
			fmt.Printf("CallTool error: %v\n", err)
		} else if callRes.IsError {
			fmt.Printf("Tool indicated an error. Content: %+v\n", callRes.Content)
		} else {
			fmt.Printf("Tool call succeeded. Content: %+v\n", callRes.Content)
		}
	}

	// Set up OS signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Press Ctrl+C to stop.")

	// Wait for termination signal
	<-sigCh
	fmt.Println("\nShutting down...")
	c.Close()
}
