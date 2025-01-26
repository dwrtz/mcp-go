package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/dwrtz/mcp-go/pkg/mcp/client"
	"github.com/dwrtz/mcp-go/pkg/mcp/logger"
)

func main() {
	// Command-line flag for specifying the server binary
	serverBinary := flag.String("server-binary", "", "Path to the MCP server binary to launch")
	flag.Parse()

	if *serverBinary == "" {
		fmt.Fprintln(os.Stderr, "Usage: mcp-client --server-binary=/path/to/server")
		os.Exit(1)
	}

	// Start the client and connect to the server
	ctx := context.Background()
	c, err := client.NewDefaultClient(ctx, *serverBinary, client.WithLogger(logger.NewStderrLogger("CLIENT")))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Client initialization error: %v\n", err)
		os.Exit(1)
	}

	// Initialize with the server
	if err := c.Initialize(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Client initialization error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Client connected & initialized with server!")

	//----------------------------------------------------------------------
	// DEMO 1: If the server supports Resources, do a resource listing.
	//----------------------------------------------------------------------
	if c.SupportsResources() {
		resources, err := c.ListResources(ctx)
		if err != nil {
			fmt.Printf("ListResources error: %v\n", err)
		} else {
			fmt.Println("Resources from server:")
			for _, r := range resources {
				fmt.Printf("  URI=%s  Name=%s\n", r.URI, r.Name)
			}
		}
	}

	//----------------------------------------------------------------------
	// DEMO 2: If the server supports Prompts, list them.
	//----------------------------------------------------------------------
	if c.SupportsPrompts() {
		prompts, err := c.ListPrompts(ctx)
		if err != nil {
			fmt.Printf("ListPrompts error: %v\n", err)
		} else {
			fmt.Println("Prompts from server:")
			for _, p := range prompts {
				fmt.Printf("  Name=%s  Description=%s\n", p.Name, p.Description)
			}
		}
	}

	//----------------------------------------------------------------------
	// DEMO 3: If the server supports Tools, we explicitly call `echo_tool`.
	//----------------------------------------------------------------------
	if c.SupportsTools() {
		fmt.Println("Server supports Tools. Calling 'echo_tool'...")

		callRes, err := c.CallTool(ctx, "echo_tool", map[string]interface{}{
			"value": "Hello from the client",
		})
		if err != nil {
			fmt.Printf("CallTool error: %v\n", err)
		} else if callRes.IsError {
			fmt.Printf("Tool indicated an error. Content: %+v\n", callRes.Content)
		} else {
			fmt.Printf("Tool call succeeded. Content: %+v\n", callRes.Content)
		}
	}

	fmt.Println("Press Ctrl+C to stop.")

	// Wait indefinitely (or do your interactive logic).
	select {}
}
