package client

import (
	"context"
	"encoding/json"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// ToolsClient provides client-side tool functionality
type ToolsClient struct {
	base *base.Client
}

// NewToolsClient creates a new ToolsClient
func NewToolsClient(base *base.Client) *ToolsClient {
	return &ToolsClient{base: base}
}

// List requests the list of available tools
func (c *ToolsClient) List(ctx context.Context) ([]types.Tool, error) {
	req := &types.ListToolsRequest{
		Method: methods.ListTools,
	}

	resp, err := c.base.SendRequest(ctx, methods.ListTools, req)
	if err != nil {
		return nil, err
	}

	var result types.ListToolsResult
	if err := json.Unmarshal(*resp.Result, &result); err != nil {
		return nil, err
	}

	return result.Tools, nil
}

// Call invokes a specific tool
func (c *ToolsClient) Call(ctx context.Context, name string, arguments map[string]interface{}) (*types.CallToolResult, error) {
	req := &types.CallToolRequest{
		Method:    methods.CallTool,
		Name:      name,
		Arguments: arguments,
	}

	resp, err := c.base.SendRequest(ctx, methods.CallTool, req)
	if err != nil {
		return nil, err
	}

	var result types.CallToolResult
	if err := json.Unmarshal(*resp.Result, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// OnToolListChanged registers a callback for tool list change notifications
func (c *ToolsClient) OnToolListChanged(callback func()) {
	c.base.RegisterNotificationHandler(methods.ToolsChanged, func(ctx context.Context, params json.RawMessage) {
		callback()
	})
}
