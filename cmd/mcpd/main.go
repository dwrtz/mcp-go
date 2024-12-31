package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dwrtz/mcp-go/internal/server"
	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/internal/transport/stdio"
)

func main() {
	// Create a new server transport using the built-in StdioTransport.
	// Attach our server.Handler to handle "ping" or unknown methods.
	srvTransport := stdio.NewStdioTransport(&transport.Options{
		Handler: &server.Handler{},
	})

	// Create the server
	srv := server.NewServer(srvTransport)

	// Set up cancellation context so we can gracefully stop on Ctrl+C
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Watch for Ctrl+C or SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "Shutting down server...")
		srv.Close()
		cancel()
	}()

	// Start the server
	fmt.Fprintln(os.Stderr, "Server starting (stdio). Press Ctrl+C to exit.")
	if err := srv.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error running server: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "Server stopped.")
}
