package types

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
)

// ToolInputSchema represents the input schema for a tool
type ToolInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// Tool represents a tool that can be called by the client
type Tool struct {
	// Name of the tool
	Name string `json:"name"`

	// Optional description
	Description string `json:"description,omitempty"`

	// JSON Schema defining expected parameters
	InputSchema ToolInputSchema `json:"inputSchema"`
}

// ListToolsRequest represents a request to list available tools
type ListToolsRequest struct {
	Method string  `json:"method"`
	Cursor *Cursor `json:"cursor,omitempty"`
}

// ListToolsResult represents the response to a tools/list request
type ListToolsResult struct {
	Tools      []Tool  `json:"tools"`
	NextCursor *Cursor `json:"nextCursor,omitempty"`
}

// CallToolRequest represents a request to call a specific tool
type CallToolRequest struct {
	Method    string                 `json:"method"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CallToolResult represents the response from a tool call
type CallToolResult struct {
	Content []interface{} `json:"content"` // Can be TextContent, ImageContent, or EmbeddedResource
	IsError bool          `json:"isError,omitempty"`
}

// ToolListChangedNotification represents a notification that the tool list has changed
type ToolListChangedNotification struct {
	Method string `json:"method"`
}

// ToolHandler is a function that handles tool invocations
type ToolHandler func(ctx context.Context, arguments map[string]interface{}) (*CallToolResult, error)

// Handler is a function that processes a tool's input and returns a result
type TypedToolHandler[T any] func(ctx context.Context, input T) (*CallToolResult, error)

// McpTool defines the interface for a typed MCP tool
type McpTool interface {
	GetName() string
	GetDescription() string
	GetDefinition() Tool
	GetHandler() ToolHandler
}

// TypedTool is a generic implementation of McpTool
type TypedTool[T any] struct {
	name        string
	description string
	handler     TypedToolHandler[T]
}

// NewTool creates a new typed MCP tool
func NewTool[T any](name, description string, handler TypedToolHandler[T]) *TypedTool[T] {
	return &TypedTool[T]{
		name:        name,
		description: description,
		handler:     handler,
	}
}

func (t *TypedTool[T]) GetName() string {
	return t.name
}

func (t *TypedTool[T]) GetDescription() string {
	return t.description
}

func (t *TypedTool[T]) GetDefinition() Tool {
	// Generate JSON schema from the type T
	reflector := &jsonschema.Reflector{
		RequiredFromJSONSchemaTags: true,
		DoNotReference:             true,
	}

	schema := reflector.Reflect(new(T))

	// Convert the orderedmap to a map[string]interface{}
	props := make(map[string]interface{})
	for pair := schema.Properties.Oldest(); pair != nil; pair = pair.Next() {
		props[pair.Key] = pair.Value
	}

	return Tool{
		Name:        t.name,
		Description: t.description,
		InputSchema: ToolInputSchema{
			Type:       "object",
			Properties: props,
			Required:   schema.Required,
		},
	}
}

func (t *TypedTool[T]) GetHandler() ToolHandler {
	return func(ctx context.Context, arguments map[string]interface{}) (*CallToolResult, error) {
		// Convert the arguments map to the typed input
		inputBytes, err := json.Marshal(arguments)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal arguments: %w", err)
		}

		var input T
		if err := json.Unmarshal(inputBytes, &input); err != nil {
			return nil, fmt.Errorf("failed to unmarshal arguments into input type: %w", err)
		}

		// Call the typed handler
		return t.handler(ctx, input)
	}
}
