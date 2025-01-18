package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// ToolsServer provides server-side tool functionality
type ToolsServer struct {
	base *base.Server
	mu   sync.RWMutex

	tools        []types.Tool
	toolHandlers map[string]ToolHandler
}

// ToolHandler is a function that handles tool invocations
type ToolHandler func(ctx context.Context, arguments map[string]interface{}) (*types.CallToolResult, error)

// NewToolsServer creates a new ToolsServer
func NewToolsServer(base *base.Server) *ToolsServer {
	s := &ToolsServer{
		base:         base,
		toolHandlers: make(map[string]ToolHandler),
	}

	// Register request handlers
	base.RegisterRequestHandler(methods.ListTools, s.handleListTools)
	base.RegisterRequestHandler(methods.CallTool, s.handleCallTool)

	return s
}

// SetTools updates the list of available tools
func (s *ToolsServer) SetTools(ctx context.Context, tools []types.Tool) error {
	s.mu.Lock()
	s.tools = tools
	s.mu.Unlock()

	return s.base.SendNotification(ctx, methods.ToolsChanged, nil)
}

// RegisterToolHandler registers a handler for a specific tool
func (s *ToolsServer) RegisterToolHandler(name string, handler ToolHandler) {
	s.mu.Lock()
	s.toolHandlers[name] = handler
	s.mu.Unlock()
}

func (s *ToolsServer) handleListTools(ctx context.Context, params json.RawMessage) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &types.ListToolsResult{
		Tools: s.tools,
	}, nil
}

func (s *ToolsServer) handleCallTool(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req types.CallToolRequest
	if err := json.Unmarshal(params, &req); err != nil {
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
