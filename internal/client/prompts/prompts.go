package prompts

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// PromptsClient provides client-side prompt functionality
type PromptsClient struct {
	base *base.Base
}

// NewPromptsClient creates a new PromptsClient
func NewPromptsClient(base *base.Base) *PromptsClient {
	return &PromptsClient{base: base}
}

// List requests the list of available prompts
func (c *PromptsClient) List(ctx context.Context) ([]types.Prompt, error) {
	req := &types.ListPromptsRequest{
		Method: methods.ListPrompts,
	}

	resp, err := c.base.SendRequest(ctx, methods.ListPrompts, req)
	if err != nil {
		return nil, err
	}

	// Check for error response
	if resp.Error != nil {
		return nil, resp.Error
	}

	// Check for nil result
	if resp.Result == nil {
		return nil, fmt.Errorf("empty response from server")
	}

	var result types.ListPromptsResult
	if err := json.Unmarshal(*resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Prompts, nil
}

// Get requests a specific prompt
func (c *PromptsClient) Get(ctx context.Context, name string, arguments map[string]string) (*types.GetPromptResult, error) {
	req := &types.GetPromptRequest{
		Method:    methods.GetPrompt,
		Name:      name,
		Arguments: arguments,
	}

	resp, err := c.base.SendRequest(ctx, methods.GetPrompt, req)
	if err != nil {
		return nil, err
	}

	// Check for error response
	if resp.Error != nil {
		return nil, resp.Error
	}

	// Check for nil result
	if resp.Result == nil {
		return nil, fmt.Errorf("empty response from server")
	}

	var result types.GetPromptResult
	if err := json.Unmarshal(*resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse prompt result: %w", err)
	}

	return &result, nil
}

// OnPromptListChanged registers a callback for prompt list change notifications
func (c *PromptsClient) OnPromptListChanged(callback func()) {
	c.base.RegisterNotificationHandler(methods.PromptsChanged, func(ctx context.Context, params json.RawMessage) {
		callback()
	})
}
