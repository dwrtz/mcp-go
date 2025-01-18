package types

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
	Role       Role        `json:"role"`
	Content    interface{} `json:"content"` // Can be TextContent or ImageContent
	Model      string      `json:"model"`
	StopReason string      `json:"stopReason,omitempty"`
}

// SamplingMessage represents a message in a sampling request
type SamplingMessage struct {
	Role    Role        `json:"role"`
	Content interface{} `json:"content"` // Can be TextContent or ImageContent
}
