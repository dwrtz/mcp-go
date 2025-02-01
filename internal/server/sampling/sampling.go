package sampling

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// Server provides server-side sampling functionality
type Server struct {
	base *base.Base
}

// NewServer creates a new Server
func NewServer(base *base.Base) *Server {
	return &Server{base: base}
}

// CreateMessage requests a sample from the language model
func (s *Server) CreateMessage(ctx context.Context, req *types.CreateMessageRequest) (*types.CreateMessageResult, error) {
	resp, err := s.base.SendRequest(ctx, methods.SampleCreate, req)
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
