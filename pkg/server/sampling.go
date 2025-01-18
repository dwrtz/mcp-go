package server

import (
	"context"
	"encoding/json"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// SamplingServer provides server-side sampling functionality
type SamplingServer struct {
	base *base.Server
}

// NewSamplingServer creates a new SamplingServer
func NewSamplingServer(base *base.Server) *SamplingServer {
	s := &SamplingServer{
		base: base,
	}

	// Register request handler for sampling/createMessage
	base.RegisterRequestHandler(methods.SampleCreate, s.handleCreateMessage)

	return s
}

func (s *SamplingServer) handleCreateMessage(ctx context.Context, params json.RawMessage) (interface{}, error) {
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
	result := &types.CreateMessageResult{
		Role:       types.RoleAssistant,
		Content:    types.TextContent{Type: "text", Text: "Sample response"},
		Model:      "sample-model",
		StopReason: "endTurn",
	}

	return result, nil
}
