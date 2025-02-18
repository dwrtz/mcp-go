package types

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	// LatestProtocolVersion represents the latest supported MCP version
	LatestProtocolVersion = "2024-11-05"

	// JSONRPCVersion represents the JSON-RPC version used by MCP
	JSONRPCVersion = "2.0"
)

// Role represents the sender or recipient in a conversation
type Role string

const (
	// RoleUser represents a user in an LLM conversation
	RoleUser Role = "user"

	// RoleAssistant represents an LLM assistant in an LLM conversation
	RoleAssistant Role = "assistant"
)

// ID represents a unique identifier for a request in JSON-RPC
type ID = jsonrpc2.ID // This is typically a string or number

// ProgressToken represents a token for tracking progress of long-running operations
type ProgressToken interface{} // string or number

// Cursor represents an opaque token for pagination
type Cursor string

// Request represents a base MCP request
type Request struct {
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params,omitempty"`
	Meta   *RequestMeta     `json:"_meta,omitempty"`
}

// RequestMeta contains metadata for requests
type RequestMeta struct {
	ProgressToken ProgressToken `json:"progressToken,omitempty"`
}

// Notification represents a base MCP notification
type Notification struct {
	Method string            `json:"method"`
	Params *json.RawMessage  `json:"params,omitempty"`
	Meta   *NotificationMeta `json:"_meta,omitempty"`
}

// NotificationMeta contains metadata for notifications
type NotificationMeta map[string]interface{}

// Result represents a base MCP result
type Result struct {
	Meta *ResultMeta `json:"_meta,omitempty"`
}

// ResultMeta contains metadata for results
type ResultMeta map[string]interface{}

// Implementation describes the name and version of an MCP implementation
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Message represents either a Request, Notification, or Response
type Message struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *ID              `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  *json.RawMessage `json:"params,omitempty"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *ErrorResponse   `json:"error,omitempty"`
}

// ErrorResponse represents a JSON-RPC 2.0 error response
type ErrorResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewError creates a new ErrorResponse with the given code and message
func NewError(code int, message string, data ...interface{}) *ErrorResponse {
	err := &ErrorResponse{
		Code:    code,
		Message: message,
	}
	if len(data) > 0 {
		err.Data = data[0]
	}
	return err
}

// Error implements the error interface.
func (e *ErrorResponse) Error() string {
	return e.Message
}

// Standard JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// PaginatedRequest represents a request that supports pagination
type PaginatedRequest struct {
	Cursor *Cursor `json:"cursor,omitempty"`
}

// PaginatedResult represents a paginated response
type PaginatedResult struct {
	NextCursor *Cursor `json:"nextCursor,omitempty"`
}

// Validate validates a Message according to the JSON-RPC 2.0 spec
func (m *Message) Validate() error {
	if m.JSONRPC != JSONRPCVersion {
		return errors.New("invalid jsonrpc version")
	}

	// Request or notification must have a method
	if m.Method != "" {
		if m.Result != nil || m.Error != nil {
			return errors.New("request/notification cannot have result or error")
		}
		// Notifications should not have an ID
		if m.ID == nil {
			// This is a notification
			return nil
		}
		// This is a request
		return nil
	}

	// Response must have an ID and either result or error, not both
	if m.ID == nil {
		return errors.New("response must have an ID")
	}
	if m.Result != nil && m.Error != nil {
		return errors.New("response cannot have both result and error")
	}
	if m.Result == nil && m.Error == nil {
		return errors.New("response must have either result or error")
	}

	return nil
}

// UnmarshalResult unmarshals the result into the provided interface
func (m *Message) UnmarshalResult(v interface{}) error {
	if m.Result == nil {
		return fmt.Errorf("no result to unmarshal")
	}
	return json.Unmarshal(*m.Result, v)
}

// InitializeRequest represents the initial request sent from client to server
type InitializeRequest struct {
	// The latest version of MCP that the client supports
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation     `json:"clientInfo"`
}

// InitializeResult represents the server's response to initialization
type InitializeResult struct {
	// The version of MCP that the server will use
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	// Optional instructions for using the server
	Instructions string `json:"instructions,omitempty"`
}

// InitializedNotification represents the notification sent after successful initialization
type InitializedNotification struct {
	Method string `json:"method"`
}
