package prompts

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// Server provides server-side prompt functionality
type Server struct {
	base *base.Base
	mu   sync.RWMutex

	prompts       []types.Prompt
	promptGetters map[string]PromptGetter
}

// PromptGetter is a function that returns a prompt result
type PromptGetter func(ctx context.Context, args map[string]string) (*types.GetPromptResult, error)

// NewServer creates a new Server
func NewServer(base *base.Base, initialPrompts []types.Prompt) *Server {
	s := &Server{
		base:          base,
		prompts:       initialPrompts,
		promptGetters: make(map[string]PromptGetter),
	}
	base.RegisterRequestHandler(methods.ListPrompts, s.handleListPrompts)
	base.RegisterRequestHandler(methods.GetPrompt, s.handleGetPrompt)
	return s
}

// SetPrompts updates the list of available prompts
func (s *Server) SetPrompts(ctx context.Context, prompts []types.Prompt) error {
	s.mu.Lock()
	s.prompts = prompts
	s.mu.Unlock()

	if s.base.Started {
		return s.base.SendNotification(ctx, methods.PromptsChanged, nil)
	}
	return nil
}

// RegisterPromptGetter registers a handler for getting prompt contents
func (s *Server) RegisterPromptGetter(name string, getter PromptGetter) {
	s.mu.Lock()
	s.promptGetters[name] = getter
	s.mu.Unlock()
}

func (s *Server) handleListPrompts(ctx context.Context, params *json.RawMessage) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &types.ListPromptsResult{
		Prompts: s.prompts,
	}, nil
}

func (s *Server) handleGetPrompt(ctx context.Context, params *json.RawMessage) (interface{}, error) {
	if params == nil {
		return nil, types.NewError(types.InvalidParams, "missing params")
	}

	var req types.GetPromptRequest
	if err := json.Unmarshal(*params, &req); err != nil {
		return nil, err
	}

	s.mu.RLock()
	getter, exists := s.promptGetters[req.Name]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no prompt found with name: %s", req.Name)
	}

	return getter(ctx, req.Arguments)
}
