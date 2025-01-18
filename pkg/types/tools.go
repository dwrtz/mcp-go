package types

// Tool represents a tool that can be called by the client
type Tool struct {
	// Name of the tool
	Name string `json:"name"`

	// Optional description
	Description string `json:"description,omitempty"`

	// JSON Schema defining expected parameters
	InputSchema struct {
		Type       string                 `json:"type"`
		Properties map[string]interface{} `json:"properties,omitempty"`
		Required   []string               `json:"required,omitempty"`
	} `json:"inputSchema"`
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
