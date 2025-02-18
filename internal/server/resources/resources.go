package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// Server provides server-side resource functionality
type Server struct {
	base *base.Base
	mu   sync.RWMutex

	resources       []types.Resource
	templates       []types.ResourceTemplate
	subscriptions   map[string][]string // URI -> subscriber IDs
	contentHandlers map[string]ContentHandler
}

// ContentHandler is a function that returns the contents of a resource
type ContentHandler func(ctx context.Context, uri string) ([]types.ResourceContent, error)

// NewServer creates a new Server
func NewServer(base *base.Base, initialResources []types.Resource, initialTemplates []types.ResourceTemplate) *Server {
	s := &Server{
		base:            base,
		resources:       initialResources,
		templates:       initialTemplates,
		subscriptions:   make(map[string][]string),
		contentHandlers: make(map[string]ContentHandler),
	}

	// Register request handlers
	base.RegisterRequestHandler(methods.ListResources, s.handleListResources)
	base.RegisterRequestHandler(methods.ReadResource, s.handleReadResource)
	base.RegisterRequestHandler(methods.ListResourceTemplates, s.handleListTemplates)
	base.RegisterRequestHandler(methods.SubscribeResource, s.handleSubscribe)
	base.RegisterRequestHandler(methods.UnsubscribeResource, s.handleUnsubscribe)

	return s
}

// SetResources updates the list of available resources
func (s *Server) SetResources(ctx context.Context, resources []types.Resource) error {
	s.mu.Lock()
	s.resources = resources
	s.mu.Unlock()

	if s.base.Started {
		return s.base.SendNotification(ctx, methods.ResourceListChanged, nil)
	}
	return nil
}

// SetTemplates updates the list of resource templates
func (s *Server) SetTemplates(ctx context.Context, templates []types.ResourceTemplate) {
	s.mu.Lock()
	s.templates = templates
	s.mu.Unlock()
}

// RegisterContentHandler registers a handler for reading resource contents
func (s *Server) RegisterContentHandler(uriPrefix string, handler ContentHandler) {
	s.mu.Lock()
	s.contentHandlers[uriPrefix] = handler
	s.mu.Unlock()
}

// NotifyResourceUpdated notifies subscribers that a resource has changed
func (s *Server) NotifyResourceUpdated(ctx context.Context, uri string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.subscriptions[uri]; exists {
		notif := &types.ResourceUpdatedNotification{
			Method: methods.ResourceUpdated,
			URI:    uri,
		}
		return s.base.SendNotification(ctx, methods.ResourceUpdated, notif)
	}
	return nil
}

func (s *Server) handleListResources(ctx context.Context, params *json.RawMessage) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &types.ListResourcesResult{
		Resources: s.resources,
	}, nil
}

func (s *Server) handleReadResource(ctx context.Context, params *json.RawMessage) (interface{}, error) {
	if params == nil {
		return nil, types.NewError(types.InvalidParams, "missing params")
	}

	var req types.ReadResourceRequest
	if err := json.Unmarshal(*params, &req); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Find matching content handler
	for prefix, handler := range s.contentHandlers {
		if len(req.URI) >= len(prefix) && req.URI[:len(prefix)] == prefix {
			contents, err := handler(ctx, req.URI)
			if err != nil {
				return nil, err
			}
			return &types.ReadResourceResult{
				Contents: contents,
			}, nil
		}
	}

	return nil, fmt.Errorf("no handler found for URI: %s", req.URI)
}

func (s *Server) handleListTemplates(ctx context.Context, params *json.RawMessage) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &types.ListResourceTemplatesResult{
		ResourceTemplates: s.templates,
	}, nil
}

func (s *Server) handleSubscribe(ctx context.Context, params *json.RawMessage) (interface{}, error) {
	if params == nil {
		return nil, types.NewError(types.InvalidParams, "missing params")
	}

	var req types.SubscribeRequest
	if err := json.Unmarshal(*params, &req); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.subscriptions[req.URI] = append(s.subscriptions[req.URI], "client-id") // TODO: Implement proper client ID tracking
	return &struct{}{}, nil
}

func (s *Server) handleUnsubscribe(ctx context.Context, params *json.RawMessage) (interface{}, error) {
	if params == nil {
		return nil, types.NewError(types.InvalidParams, "missing params")
	}

	var req types.UnsubscribeRequest
	if err := json.Unmarshal(*params, &req); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.subscriptions, req.URI)
	return &struct{}{}, nil
}
