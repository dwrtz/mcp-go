package mcp

import (
	"context"
	"fmt"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/client"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// Client represents a Model Context Protocol client
type Client struct {
	base *base.Client

	// Feature-specific clients
	roots     *client.RootsClient
	resources *client.ResourcesClient
	prompts   *client.PromptsClient

	// Client capabilities
	capabilities types.ClientCapabilities
}

// NewClient creates a new MCP client
func NewClient(transport transport.Transport) *Client {
	return &Client{
		base: base.NewClient(transport),
		capabilities: types.ClientCapabilities{
			Roots: &types.RootsClientCapabilities{
				ListChanged: true,
			},
			Sampling: &types.SamplingClientCapabilities{},
		},
	}
}

// Initialize initiates the connection with the server
func (c *Client) Initialize(ctx context.Context) error {
	// Create initialization request
	req := &types.InitializeRequest{
		ProtocolVersion: types.LatestProtocolVersion,
		Capabilities:    c.capabilities,
		ClientInfo: types.Implementation{
			Name:    "mcp-go",
			Version: "0.1.0",
		},
	}

	// Send initialize request
	resp, err := c.base.SendRequest(ctx, methods.Initialize, req)
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	// Parse server response
	var result types.InitializeResult
	if err := resp.UnmarshalResult(&result); err != nil {
		return fmt.Errorf("failed to parse initialization response: %w", err)
	}

	// Verify protocol version compatibility
	if result.ProtocolVersion != types.LatestProtocolVersion {
		return fmt.Errorf("server protocol version %s not supported", result.ProtocolVersion)
	}

	// Initialize feature-specific clients based on server capabilities
	if result.Capabilities.Resources != nil {
		c.roots = client.NewRootsClient(c.base)
		c.resources = client.NewResourcesClient(c.base)
	}

	if result.Capabilities.Prompts != nil {
		c.prompts = client.NewPromptsClient(c.base)
	}

	// Send initialized notification
	if err := c.base.SendNotification(ctx, methods.Initialized, nil); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	return nil
}

// Start begins processing messages
func (c *Client) Start(ctx context.Context) error {
	return c.base.Start(ctx)
}

// Close shuts down the client
func (c *Client) Close() error {
	return c.base.Close()
}

// SupportsRoots returns whether the server supports roots functionality
func (c *Client) SupportsRoots() bool {
	return c.roots != nil
}

// SupportsResources returns whether the server supports resources functionality
func (c *Client) SupportsResources() bool {
	return c.resources != nil
}

// SupportsPrompts returns whether the server supports prompts functionality
func (c *Client) SupportsPrompts() bool {
	return c.prompts != nil
}

// Roots returns the roots client if the server supports it
func (c *Client) Roots() *client.RootsClient {
	return c.roots
}

// Resources returns the resources client if the server supports it
func (c *Client) Resources() *client.ResourcesClient {
	return c.resources
}

// Prompts returns the prompts client if the server supports it
func (c *Client) Prompts() *client.PromptsClient {
	return c.prompts
}
