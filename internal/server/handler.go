package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dwrtz/mcp-go/pkg/types"
)

// Handler implements the transport.Handler interface.
type Handler struct {
	Logger interface {
		Logf(format string, args ...interface{})
	}
}

// Handle processes incoming messages and optionally returns a response.
func (h *Handler) Handle(ctx context.Context, msg *types.Message) (*types.Message, error) {
	if h.Logger != nil {
		h.Logger.Logf("Server received message: %+v", msg)
	}

	switch msg.Method {
	case "ping":
		// Ensure we send back the exact same ID we received
		response := &types.Message{
			JSONRPC: types.JSONRPCVersion,
			ID:      msg.ID, // Important: preserve the original ID
			Result:  rawJSON(`{"status":"ok"}`),
		}
		if h.Logger != nil {
			h.Logger.Logf("Server sending ping response with ID %v", msg.ID)
		}
		return response, nil

	default:
		return nil, fmt.Errorf("unknown method: %s", msg.Method)
	}
}

func rawJSON(s string) *json.RawMessage {
	raw := json.RawMessage(s)
	return &raw
}
