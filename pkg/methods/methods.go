package methods

// Method constants for basic protocol operations
const (
	// Initialization methods
	Initialize  = "initialize"
	Initialized = "notifications/initialized"

	// Utility methods
	Ping      = "ping"
	Cancelled = "notifications/cancelled"
	Progress  = "notifications/progress"
	Message   = "notifications/message" // For logging

	// Client methods
	ListRoots    = "roots/list"
	RootsChanged = "notifications/roots/list_changed"
	SampleCreate = "sampling/createMessage"

	// Server methods - Resources
	ListResources         = "resources/list"
	ReadResource          = "resources/read"
	ListResourceTemplates = "resources/templates/list"
	SubscribeResource     = "resources/subscribe"
	UnsubscribeResource   = "resources/unsubscribe"
	ResourceUpdated       = "notifications/resources/updated"
	ResourceListChanged   = "notifications/resources/list_changed"

	// Server methods - Prompts
	ListPrompts    = "prompts/list"
	GetPrompt      = "prompts/get"
	PromptsChanged = "notifications/prompts/list_changed"

	// Server methods - Tools
	ListTools    = "tools/list"
	CallTool     = "tools/call"
	ToolsChanged = "notifications/tools/list_changed"

	// Server methods - Logging
	SetLogLevel = "logging/setLevel"

	// Server methods - Completion
	Complete = "completion/complete"
)
