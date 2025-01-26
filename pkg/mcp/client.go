package mcp

import (
	"context"
	"fmt"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/internal/client/prompts"
	"github.com/dwrtz/mcp-go/internal/client/resources"
	"github.com/dwrtz/mcp-go/internal/client/roots"
	"github.com/dwrtz/mcp-go/internal/client/sampling"
	"github.com/dwrtz/mcp-go/internal/client/tools"
	"github.com/dwrtz/mcp-go/internal/transport"
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
func WithRoots(initialRoots []types.Root) ClientOption {
	return func(c *Client) {
		c.capabilities.Roots = &types.RootsClientCapabilities{
			ListChanged: true,
		}
		c.roots = roots.NewRootsClient(c.base, initialRoots)
	}
}

// WithSampling enables sampling functionality on the client
func WithSampling(handler types.SamplingHandler) ClientOption {
	return func(c *Client) {
		c.capabilities.Sampling = &types.SamplingClientCapabilities{}
		c.sampling = sampling.NewSamplingClient(c.base, handler)
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
		c.OnResourceListChanged(func() {
			// default noop
			c.base.Logf("from server: %s", methods.ResourceListChanged)
		})
		c.OnResourceUpdated(func(uri string) {
			// default noop
			c.base.Logf("from server: %s %s", methods.ResourceUpdated, uri)
		})
	}

	if result.Capabilities.Prompts != nil {
		c.prompts = prompts.NewPromptsClient(c.base)
		c.OnPromptListChanged(func() {
			// default noop
			c.base.Logf("from server: %s", methods.PromptsChanged)
		})
	}

	if result.Capabilities.Tools != nil {
		c.tools = tools.NewToolsClient(c.base)
		c.OnToolListChanged(func() {
			// default noop
			c.base.Logf("from server: %s", methods.ToolsChanged)
		})
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

// SupportsRoots returns whether the client supports roots functionality
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

// Resource Methods

// ListResources returns a list of all available resources from the server.
// Returns an error if the server does not support resources.
func (c *Client) ListResources(ctx context.Context) ([]types.Resource, error) {
	if !c.SupportsResources() {
		return nil, types.NewError(types.MethodNotFound, "resources not supported")
	}
	return c.resources.List(ctx)
}

// ReadResource retrieves the contents of a specific resource identified by its URI.
// Returns the resource contents, which can be either text or binary data.
// Returns an error if the server does not support resources or if the resource cannot be read.
func (c *Client) ReadResource(ctx context.Context, uri string) ([]types.ResourceContent, error) {
	if !c.SupportsResources() {
		return nil, types.NewError(types.MethodNotFound, "resources not supported")
	}
	return c.resources.Read(ctx, uri)
}

// ListResourceTemplates returns a list of available resource templates from the server.
// Templates can be used to construct valid resource URIs.
// Returns an error if the server does not support resources.
func (c *Client) ListResourceTemplates(ctx context.Context) ([]types.ResourceTemplate, error) {
	if !c.SupportsResources() {
		return nil, types.NewError(types.MethodNotFound, "resources not supported")
	}
	return c.resources.ListTemplates(ctx)
}

// SubscribeResource subscribes to updates for a specific resource identified by its URI.
// The client will receive notifications through OnResourceUpdated when the resource changes.
// Returns an error if the server does not support resources or subscriptions.
func (c *Client) SubscribeResource(ctx context.Context, uri string) error {
	if !c.SupportsResources() {
		return types.NewError(types.MethodNotFound, "resources not supported")
	}
	return c.resources.Subscribe(ctx, uri)
}

// UnsubscribeResource removes a subscription for a specific resource.
// Returns an error if the server does not support resources or if the subscription cannot be removed.
func (c *Client) UnsubscribeResource(ctx context.Context, uri string) error {
	if !c.SupportsResources() {
		return types.NewError(types.MethodNotFound, "resources not supported")
	}
	return c.resources.Unsubscribe(ctx, uri)
}

// OnResourceUpdated registers a callback that will be invoked when a subscribed resource changes.
// The callback receives the URI of the updated resource.
// No-op if the server does not support resources.
func (c *Client) OnResourceUpdated(callback func(uri string)) {
	if c.SupportsResources() {
		c.resources.OnResourceUpdated(callback)
	}
}

// OnResourceListChanged registers a callback that will be invoked when the list of available
// resources changes on the server. No-op if the server does not support resources.
func (c *Client) OnResourceListChanged(callback func()) {
	if c.SupportsResources() {
		c.resources.OnResourceListChanged(callback)
	}
}

// Prompt Methods

// ListPrompts returns a list of all available prompts from the server.
// Returns an error if the server does not support prompts.
func (c *Client) ListPrompts(ctx context.Context) ([]types.Prompt, error) {
	if !c.SupportsPrompts() {
		return nil, types.NewError(types.MethodNotFound, "prompts not supported")
	}
	return c.prompts.List(ctx)
}

// GetPrompt retrieves a specific prompt by name, with optional arguments for templating.
// Returns the prompt content and any associated messages.
// Returns an error if the server does not support prompts or if the prompt cannot be found.
func (c *Client) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*types.GetPromptResult, error) {
	if !c.SupportsPrompts() {
		return nil, types.NewError(types.MethodNotFound, "prompts not supported")
	}
	return c.prompts.Get(ctx, name, arguments)
}

// OnPromptListChanged registers a callback that will be invoked when the list of available
// prompts changes on the server. No-op if the server does not support prompts.
func (c *Client) OnPromptListChanged(callback func()) {
	if c.SupportsPrompts() {
		c.prompts.OnPromptListChanged(callback)
	}
}

// Tool Methods

// ListTools returns a list of all available tools from the server.
// Returns an error if the server does not support tools.
func (c *Client) ListTools(ctx context.Context) ([]types.Tool, error) {
	if !c.SupportsTools() {
		return nil, types.NewError(types.MethodNotFound, "tools not supported")
	}
	return c.tools.List(ctx)
}

// CallTool invokes a specific tool by name with the provided arguments.
// Returns the tool's execution result or an error if the tool cannot be called.
// Returns an error if the server does not support tools.
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*types.CallToolResult, error) {
	if !c.SupportsTools() {
		return nil, types.NewError(types.MethodNotFound, "tools not supported")
	}
	return c.tools.Call(ctx, name, arguments)
}

// OnToolListChanged registers a callback that will be invoked when the list of available
// tools changes on the server. No-op if the server does not support tools.
func (c *Client) OnToolListChanged(callback func()) {
	if c.SupportsTools() {
		c.tools.OnToolListChanged(callback)
	}
}

// Root Methods

// SetRoots updates the list of root directories that the client exposes to the server.
// Each root must be a valid file:// URI.
// Returns an error if the client does not support roots or if any root is invalid.
func (c *Client) SetRoots(ctx context.Context, roots []types.Root) error {
	if !c.SupportsRoots() {
		return types.NewError(types.MethodNotFound, "roots not supported")
	}
	return c.roots.SetRoots(ctx, roots)
}
