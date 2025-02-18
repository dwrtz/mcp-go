package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// Server provides server-side tool functionality
type Server struct {
	base *base.Base
	mu   sync.RWMutex

	tools        []types.Tool
	toolHandlers map[string]types.ToolHandler
}

// NewServer creates a new Server
func NewServer(base *base.Base, initialTools []types.McpTool) *Server {
	var newTools []types.Tool
	newToolHandlers := make(map[string]types.ToolHandler)

	for _, tool := range initialTools {
		newTools = append(newTools, tool.GetDefinition())
		newToolHandlers[tool.GetName()] = tool.GetHandler()
	}

	s := &Server{
		base:         base,
		tools:        newTools,
		toolHandlers: newToolHandlers,
	}
	base.RegisterRequestHandler(methods.ListTools, s.handleListTools)
	base.RegisterRequestHandler(methods.CallTool, s.handleCallTool)
	return s
}

// SetTools updates the list of available tools
func (s *Server) SetTools(ctx context.Context, tools []types.McpTool) error {
	var newTools []types.Tool
	newToolHandlers := make(map[string]types.ToolHandler)

	for _, tool := range tools {
		newTools = append(newTools, tool.GetDefinition())
		newToolHandlers[tool.GetName()] = tool.GetHandler()
	}

	s.mu.Lock()
	s.tools = newTools
	s.toolHandlers = newToolHandlers
	s.mu.Unlock()

	if s.base.Started {
		return s.base.SendNotification(ctx, methods.ToolsChanged, nil)
	}
	return nil
}

func (s *Server) handleListTools(ctx context.Context, params *json.RawMessage) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &types.ListToolsResult{
		Tools: s.tools,
	}, nil
}

func (s *Server) handleCallTool(ctx context.Context, params *json.RawMessage) (interface{}, error) {
	if params == nil {
		return nil, types.NewError(types.InvalidParams, "missing params")
	}

	var req types.CallToolRequest
	if err := json.Unmarshal(*params, &req); err != nil {
		return nil, err
	}

	s.mu.RLock()
	handler, exists := s.toolHandlers[req.Name]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no handler found for tool: %s", req.Name)
	}

	return handler(ctx, req.Arguments)
}
