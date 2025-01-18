package types

// Prompt represents a prompt or prompt template
type Prompt struct {
	// Name uniquely identifies the prompt
	Name string `json:"name"`

	// Optional description
	Description string `json:"description,omitempty"`

	// Optional arguments for templating
	Arguments []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument describes an argument a prompt can accept
type PromptArgument struct {
	// Name of the argument
	Name string `json:"name"`

	// Optional description
	Description string `json:"description,omitempty"`

	// Whether this argument is required
	Required bool `json:"required,omitempty"`
}

// PromptMessage represents a message in a prompt
type PromptMessage struct {
	Role    Role        `json:"role"`
	Content interface{} `json:"content"` // Can be TextContent, ImageContent, or EmbeddedResource
}

// TextContent represents text provided to/from an LLM
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ImageContent represents an image provided to/from an LLM
type ImageContent struct {
	Type     string `json:"type"`
	Data     string `json:"data"` // base64-encoded
	MimeType string `json:"mimeType"`
}

// EmbeddedResource represents a resource embedded in a prompt
type EmbeddedResource struct {
	Type     string      `json:"type"`
	Resource interface{} `json:"resource"` // Can be TextResourceContents or BlobResourceContents
}

// ListPromptsRequest represents a request to list available prompts
type ListPromptsRequest struct {
	Method string  `json:"method"`
	Cursor *Cursor `json:"cursor,omitempty"`
}

// ListPromptsResult represents the response to a prompts/list request
type ListPromptsResult struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor *Cursor  `json:"nextCursor,omitempty"`
}

// GetPromptRequest represents a request to get a specific prompt
type GetPromptRequest struct {
	Method    string            `json:"method"`
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

// GetPromptResult represents the response to a prompts/get request
type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// PromptListChangedNotification represents a notification that the prompt list has changed
type PromptListChangedNotification struct {
	Method string `json:"method"`
}
