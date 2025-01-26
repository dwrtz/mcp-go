package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/internal/transport/stdio"
	"github.com/dwrtz/mcp-go/pkg/mcp"
)

func main() {
	// Command-line flag for specifying the server binary
	serverBinary := flag.String("server-binary", "", "Path to the MCP server binary to launch")
	flag.Parse()

	if *serverBinary == "" {
		fmt.Fprintln(os.Stderr, "Usage: mcp-client --server-binary=/path/to/server")
		os.Exit(1)
	}

	// Launch the server as a child process, hooking up stdio
	cmd := exec.Command(*serverBinary)
	serverOut, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create stdout pipe for server: %v\n", err)
		os.Exit(1)
	}
	serverIn, err := cmd.StdinPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create stdin pipe for server: %v\n", err)
		os.Exit(1)
	}
	// Optionally pipe server stderr to our own stderr
	cmd.Stderr = os.Stderr

	fmt.Printf("Launching server: %s\n", *serverBinary)
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server process: %v\n", err)
		os.Exit(1)
	}

	// Wire up a StdioTransport that reads from serverOut, writes to serverIn
	transport := stdio.NewStdioTransport(
		io.NopCloser(serverOut), // read end
		serverIn,                // write end
		newStdioLogger("CLIENT"),
	)

	// Create a new MCP client with minimal or default capabilities.
	client := mcp.NewClient(transport)

	// Start the transport
	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Client transport start error: %v\n", err)
		os.Exit(1)
	}

	// Initialize with the server
	if err := client.Initialize(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Client initialization error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Client connected & initialized with server!")

	//----------------------------------------------------------------------
	// DEMO 1: If the server supports Resources, do a resource listing.
	//----------------------------------------------------------------------
	if client.SupportsResources() {
		resources, err := client.ListResources(ctx)
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
	if client.SupportsPrompts() {
		prompts, err := client.ListPrompts(ctx)
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
	if client.SupportsTools() {
		fmt.Println("Server supports Tools. Calling 'echo_tool'...")

		callRes, err := client.CallTool(ctx, "echo_tool", map[string]interface{}{
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

// newStdioLogger returns a basic transport.Logger that prints to stderr with a prefix.
func newStdioLogger(prefix string) transport.Logger {
	return &stdioLogger{prefix: prefix}
}

type stdioLogger struct {
	prefix string
}

func (l *stdioLogger) Logf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[%s] ", l.prefix)
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
