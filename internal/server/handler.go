package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dwrtz/mcp-go/pkg/types"
	"github.com/sourcegraph/jsonrpc2"
)

// Handler implements the transport.Handler interface.
// It defines how incoming requests are processed.
type Handler struct{}

// Handle processes incoming messages and optionally returns a response.
func (h *Handler) Handle(ctx context.Context, msg *types.Message) (*types.Message, error) {
	switch msg.Method {
	case "ping":
		raw := json.RawMessage(`{"message":"pong"}`)
		return &types.Message{
			JSONRPC: types.JSONRPCVersion,
			ID:      msg.ID,
			Result:  &raw,
		}, nil

	default:
		// If we don't recognize the method, return a standard JSON-RPC error
		return nil, &jsonrpc2.Error{
			Code:    types.MethodNotFound,
			Message: fmt.Sprintf("Unknown method: %s", msg.Method),
		}
	}
}
