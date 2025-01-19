package sampling

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// SamplingClient provides client-side sampling functionality
type SamplingClient struct {
	base *base.Client
}

// NewSamplingClient creates a new SamplingClient
func NewSamplingClient(base *base.Client) *SamplingClient {
	return &SamplingClient{base: base}
}

// CreateMessage requests a sample from the language model
func (c *SamplingClient) CreateMessage(ctx context.Context, req *types.CreateMessageRequest) (*types.CreateMessageResult, error) {
	resp, err := c.base.SendRequest(ctx, methods.SampleCreate, req)
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

	var result types.CreateMessageResult
	if err := json.Unmarshal(*resp.Result, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateMessageWithDefaults is a convenience method for simple sampling requests
func (c *SamplingClient) CreateMessageWithDefaults(ctx context.Context, messages []types.SamplingMessage) (*types.CreateMessageResult, error) {
	req := &types.CreateMessageRequest{
		Method:    methods.SampleCreate,
		Messages:  messages,
		MaxTokens: 1000, // Default max tokens
	}

	return c.CreateMessage(ctx, req)
}
