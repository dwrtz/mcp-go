package sampling

import (
	"context"
	"encoding/json"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// SamplingClient provides client-side sampling functionality
type SamplingClient struct {
	base *base.Base
}

// NewSamplingClient creates a new SamplingClient
func NewSamplingClient(base *base.Base) *SamplingClient {
	c := &SamplingClient{
		base: base,
	}

	// Register request handler for sampling/createMessage
	base.RegisterRequestHandler(methods.SampleCreate, c.handleCreateMessage)

	return c
}

func (c *SamplingClient) handleCreateMessage(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req types.CreateMessageRequest
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, err
	}

	// Validate request
	if len(req.Messages) == 0 {
		return nil, types.NewError(types.InvalidParams, "messages array cannot be empty")
	}

	if req.MaxTokens <= 0 {
		return nil, types.NewError(types.InvalidParams, "maxTokens must be positive")
	}

	// Process the sampling request through the client's sampling capability
	// The actual implementation would depend on the specific LLM integration
	// This is just a placeholder response
	// TODO: Implement actual sampling logic
	result := &types.CreateMessageResult{
		Role:       types.RoleAssistant,
		Content:    types.TextContent{Type: "text", Text: "Sample response"},
		Model:      "sample-model",
		StopReason: "endTurn",
	}

	return result, nil
}
