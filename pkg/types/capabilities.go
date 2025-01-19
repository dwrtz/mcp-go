package types

// ClientCapabilities represents the capabilities a client supports
type ClientCapabilities struct {
	// Experimental features support
	Experimental map[string]interface{} `json:"experimental,omitempty"`

	// Roots capability
	Roots *RootsClientCapabilities `json:"roots,omitempty"`

	// Sampling capability
	Sampling *SamplingClientCapabilities `json:"sampling,omitempty"`
}

// RootsClientCapabilities represents roots-specific client capabilities
type RootsClientCapabilities struct {
	// Whether the client supports notifications for changes to the roots list
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingClientCapabilities represents sampling-specific client capabilities
type SamplingClientCapabilities struct {
	// Currently empty as per spec, but included for future extensibility
}

// ServerCapabilities represents the capabilities a server supports
type ServerCapabilities struct {
	// Experimental features support
	Experimental map[string]interface{} `json:"experimental,omitempty"`

	// Logging capability
	Logging *LoggingServerCapabilities `json:"logging,omitempty"`

	// Prompts capability
	Prompts *PromptsServerCapabilities `json:"prompts,omitempty"`

	// Resources capability
	Resources *ResourcesServerCapabilities `json:"resources,omitempty"`

	// Tools capability
	Tools *ToolsServerCapabilities `json:"tools,omitempty"`
}

// LoggingServerCapabilities represents logging-specific server capabilities
type LoggingServerCapabilities struct {
	// Currently empty as per spec, but included for future extensibility
}

// PromptsServerCapabilities represents prompts-specific server capabilities
type PromptsServerCapabilities struct {
	// Whether the server supports notifications for changes to the prompt list
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesServerCapabilities represents resources-specific server capabilities
type ResourcesServerCapabilities struct {
	// Whether the server supports subscribing to resource updates
	Subscribe bool `json:"subscribe,omitempty"`

	// Whether the server supports notifications for changes to the resource list
	ListChanged bool `json:"listChanged,omitempty"`
}

// ToolsServerCapabilities represents tools-specific server capabilities
type ToolsServerCapabilities struct {
	// Whether the server supports notifications for changes to the tool list
	ListChanged bool `json:"listChanged,omitempty"`
}
