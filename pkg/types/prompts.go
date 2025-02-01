package types

import (
	"encoding/json"
	"fmt"
)

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
	Role    Role           `json:"role"`
	Content MessageContent `json:"content"`
}

// MessageContent is an interface that all content types must implement
type MessageContent interface {
	contentType() string
}

// TextContent represents text provided to/from an LLM
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (t TextContent) contentType() string {
	return "text"
}

// ImageContent represents an image provided to/from an LLM
type ImageContent struct {
	Type     string `json:"type"`
	Data     string `json:"data"` // base64-encoded
	MimeType string `json:"mimeType"`
}

func (i ImageContent) contentType() string {
	return "image"
}

// EmbeddedResource represents a resource embedded in a prompt
type EmbeddedResource struct {
	Type     string           `json:"type"`
	Resource ResourceContents `json:"resource"`
}

func (e EmbeddedResource) contentType() string {
	return "resource"
}

// UnmarshalJSON unmarshals a PromptMessage
func (m *PromptMessage) UnmarshalJSON(data []byte) error {
	type Alias PromptMessage // Avoid recursive unmarshaling
	aux := &struct {
		*Alias
		Content json.RawMessage `json:"content"`
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Unmarshal content based on type
	var contentType struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(aux.Content, &contentType); err != nil {
		return err
	}

	switch contentType.Type {
	case "text":
		var text TextContent
		if err := json.Unmarshal(aux.Content, &text); err != nil {
			return err
		}
		m.Content = text
	case "image":
		var img ImageContent
		if err := json.Unmarshal(aux.Content, &img); err != nil {
			return err
		}
		m.Content = img
	case "resource":
		var res EmbeddedResource
		if err := json.Unmarshal(aux.Content, &res); err != nil {
			return err
		}
		m.Content = res
	default:
		return fmt.Errorf("unknown content type: %s", contentType.Type)
	}

	return nil
}

// MarshalJSON marshals a PromptMessage
func (m PromptMessage) MarshalJSON() ([]byte, error) {
	type Alias PromptMessage
	return json.Marshal(&struct {
		Alias
		Content MessageContent `json:"content"`
	}{
		Alias:   (Alias)(m),
		Content: m.Content,
	})
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
