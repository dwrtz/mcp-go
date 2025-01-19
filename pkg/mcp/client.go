package mcp

import (
	"context"
	"fmt"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/client/prompts"
	"github.com/dwrtz/mcp-go/pkg/client/resources"
	"github.com/dwrtz/mcp-go/pkg/client/roots"
	"github.com/dwrtz/mcp-go/pkg/client/sampling"
	"github.com/dwrtz/mcp-go/pkg/client/tools"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// Client represents a Model Context Protocol client
type Client struct {
	base *base.Base

	// Feature-specific clients
	roots     *roots.RootsClient
	resources *resources.ResourcesClient
	prompts   *prompts.PromptsClient
	tools     *tools.ToolsClient
	sampling  *sampling.SamplingClient

	// Client capabilities
	capabilities types.ClientCapabilities
}

// ClientOption is a function that configures a Client
type ClientOption func(*Client)

// WithRoots enables roots functionality on the client
func WithRoots() ClientOption {
	return func(c *Client) {
		c.capabilities.Roots = &types.RootsClientCapabilities{
			ListChanged: true,
		}
	}
}

// WithSampling enables sampling functionality on the client
func WithSampling() ClientOption {
	return func(c *Client) {
		c.capabilities.Sampling = &types.SamplingClientCapabilities{}
	}
}

// NewClient creates a new MCP client
func NewClient(transport transport.Transport, opts ...ClientOption) *Client {
	c := &Client{
		base:         base.NewBase(transport),
		capabilities: types.ClientCapabilities{},
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	return c
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
		c.resources = resources.NewResourcesClient(c.base)
	}

	if result.Capabilities.Prompts != nil {
		c.prompts = prompts.NewPromptsClient(c.base)
	}

	if result.Capabilities.Tools != nil {
		c.tools = tools.NewToolsClient(c.base)
	}

	// Initialize sampling and roots client if we declared the capability
	if c.capabilities.Sampling != nil {
		c.sampling = sampling.NewSamplingClient(c.base)
	}

	if c.capabilities.Roots != nil {
		c.roots = roots.NewRootsClient(c.base)
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

// SupportsTools returns whether the server supports tools functionality
func (c *Client) SupportsTools() bool {
	return c.tools != nil
}

// SupportsSampling returns whether the client supports sampling functionality
func (c *Client) SupportsSampling() bool {
	return c.sampling != nil
}

// Roots returns the roots client
func (c *Client) Roots() *roots.RootsClient {
	return c.roots
}

// Resources returns the resources client
func (c *Client) Resources() *resources.ResourcesClient {
	return c.resources
}

// Prompts returns the prompts client
func (c *Client) Prompts() *prompts.PromptsClient {
	return c.prompts
}

// Tools returns the tools client
func (c *Client) Tools() *tools.ToolsClient {
	return c.tools
}

// Sampling returns the sampling client
func (c *Client) Sampling() *sampling.SamplingClient {
	return c.sampling
}
