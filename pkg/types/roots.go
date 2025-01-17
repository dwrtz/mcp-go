package types

// Root represents a root directory or file that the server can operate on
type Root struct {
	// URI identifying the root. Must start with file:// for now
	URI string `json:"uri"`

	// Optional name for the root
	Name string `json:"name,omitempty"`
}

// ListRootsRequest represents a request to list available roots
type ListRootsRequest struct {
	Method string `json:"method"`
}

// ListRootsResult represents the response to a roots/list request
type ListRootsResult struct {
	Roots []Root `json:"roots"`
}

// RootsListChangedNotification represents a notification that the roots list has changed
type RootsListChangedNotification struct {
	Method string `json:"method"`
}
