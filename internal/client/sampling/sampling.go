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
	base    *base.Base
	handler types.SamplingHandler
}

// NewSamplingClient creates a new SamplingClient
func NewSamplingClient(base *base.Base, handler types.SamplingHandler) *SamplingClient {
	c := &SamplingClient{
		base:    base,
		handler: handler,
	}

	// Register request handler for sampling/createMessage
	base.RegisterRequestHandler(methods.SampleCreate, c.handleCreateMessage)

	return c
}

func (c *SamplingClient) handleCreateMessage(ctx context.Context, params *json.RawMessage) (interface{}, error) {
	var req types.CreateMessageRequest
	if params == nil {
		return nil, types.NewError(types.InvalidParams, "missing params")
	}
	if err := json.Unmarshal(*params, &req); err != nil {
		return nil, err
	}
	return c.handler(ctx, &req)
}
