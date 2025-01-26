package types

import (
	"context"
	"encoding/json"
	"fmt"
)

// ModelPreferences represents server preferences for model selection
type ModelPreferences struct {
	// Optional hints for model selection
	Hints []ModelHint `json:"hints,omitempty"`

	// Priority values between 0 and 1
	CostPriority         float64 `json:"costPriority,omitempty"`
	SpeedPriority        float64 `json:"speedPriority,omitempty"`
	IntelligencePriority float64 `json:"intelligencePriority,omitempty"`
}

// ModelHint provides hints for model selection
type ModelHint struct {
	Name string `json:"name,omitempty"`
}

// CreateMessageRequest represents a request to sample from an LLM
type CreateMessageRequest struct {
	Method           string            `json:"method"`
	Messages         []SamplingMessage `json:"messages"`
	ModelPreferences *ModelPreferences `json:"modelPreferences,omitempty"`
	SystemPrompt     string            `json:"systemPrompt,omitempty"`
	IncludeContext   string            `json:"includeContext,omitempty"`
	Temperature      float64           `json:"temperature,omitempty"`
	MaxTokens        int               `json:"maxTokens"`
	StopSequences    []string          `json:"stopSequences,omitempty"`
	Metadata         interface{}       `json:"metadata,omitempty"`
}

// CreateMessageResult represents the response from a sampling request
type CreateMessageResult struct {
	Role       Role           `json:"role"`
	Content    MessageContent `json:"content"` // Using the same MessageContent interface from prompts
	Model      string         `json:"model"`
	StopReason string         `json:"stopReason,omitempty"`
}

// SamplingMessage represents a message in a sampling request
type SamplingMessage struct {
	Role    Role           `json:"role"`
	Content MessageContent `json:"content"` // Using the same MessageContent interface
}

// Custom unmarshaling for SamplingMessage
func (m *SamplingMessage) UnmarshalJSON(data []byte) error {
	type Alias SamplingMessage
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
	default:
		return fmt.Errorf("unknown content type: %s", contentType.Type)
	}

	return nil
}

// Custom marshaling for SamplingMessage
func (m SamplingMessage) MarshalJSON() ([]byte, error) {
	type Alias SamplingMessage
	return json.Marshal(&struct {
		Alias
		Content MessageContent `json:"content"`
	}{
		Alias:   (Alias)(m),
		Content: m.Content,
	})
}

// Custom unmarshaling for CreateMessageResult
func (r *CreateMessageResult) UnmarshalJSON(data []byte) error {
	type Alias CreateMessageResult
	aux := &struct {
		*Alias
		Content json.RawMessage `json:"content"`
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

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
		r.Content = text
	case "image":
		var img ImageContent
		if err := json.Unmarshal(aux.Content, &img); err != nil {
			return err
		}
		r.Content = img
	default:
		return fmt.Errorf("unknown content type: %s", contentType.Type)
	}

	return nil
}

// Custom marshaling for CreateMessageResult
func (r CreateMessageResult) MarshalJSON() ([]byte, error) {
	type Alias CreateMessageResult
	return json.Marshal(&struct {
		Alias
		Content MessageContent `json:"content"`
	}{
		Alias:   (Alias)(r),
		Content: r.Content,
	})
}

// SamplingHandler is a function that handles a sampling request
type SamplingHandler func(ctx context.Context, req *CreateMessageRequest) (*CreateMessageResult, error)
