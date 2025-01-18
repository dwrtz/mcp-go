package types

import (
	"encoding/base64"
)

// Resource represents a known resource that the server can read
type Resource struct {
	// URI identifying this resource
	URI string `json:"uri"`

	// Human-readable name
	Name string `json:"name"`

	// Optional description
	Description string `json:"description,omitempty"`

	// Optional MIME type
	MimeType string `json:"mimeType,omitempty"`
}

// ResourceContents represents the contents of a specific resource
type ResourceContents struct {
	// URI identifying this resource
	URI string `json:"uri"`

	// Optional MIME type
	MimeType string `json:"mimeType,omitempty"`
}

// TextResourceContents represents text-based resource contents
type TextResourceContents struct {
	ResourceContents
	Text string `json:"text"`
}

// BlobResourceContents represents binary resource contents
type BlobResourceContents struct {
	ResourceContents
	Blob string `json:"blob"` // base64-encoded data
}

// NewBlobContents creates a new BlobResourceContents from raw binary data
func NewBlobContents(uri string, mimeType string, data []byte) BlobResourceContents {
	return BlobResourceContents{
		ResourceContents: ResourceContents{
			URI:      uri,
			MimeType: mimeType,
		},
		Blob: base64.StdEncoding.EncodeToString(data),
	}
}

// GetData decodes the blob data
func (b *BlobResourceContents) GetData() ([]byte, error) {
	return base64.StdEncoding.DecodeString(b.Blob)
}

// ResourceTemplate represents a template for available resources
type ResourceTemplate struct {
	// URI template for constructing resource URIs
	URITemplate string `json:"uriTemplate"`

	// Human-readable name
	Name string `json:"name"`

	// Optional description
	Description string `json:"description,omitempty"`

	// Optional MIME type
	MimeType string `json:"mimeType,omitempty"`
}

// ListResourcesRequest represents a request to list available resources
type ListResourcesRequest struct {
	Method string  `json:"method"`
	Cursor *Cursor `json:"cursor,omitempty"`
}

// ListResourcesResult represents the response to a resources/list request
type ListResourcesResult struct {
	Resources  []Resource `json:"resources"`
	NextCursor *Cursor    `json:"nextCursor,omitempty"`
}

// ListResourceTemplatesRequest represents a request to list resource templates
type ListResourceTemplatesRequest struct {
	Method string  `json:"method"`
	Cursor *Cursor `json:"cursor,omitempty"`
}

// ListResourceTemplatesResult represents the response to a resources/templates/list request
type ListResourceTemplatesResult struct {
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
	NextCursor        *Cursor            `json:"nextCursor,omitempty"`
}

// ReadResourceRequest represents a request to read a specific resource
type ReadResourceRequest struct {
	Method string `json:"method"`
	URI    string `json:"uri"`
}

// ReadResourceResult represents the response to a resources/read request
type ReadResourceResult struct {
	Contents []interface{} `json:"contents"` // Can be TextResourceContents or BlobResourceContents
}

// ResourceListChangedNotification represents a notification that the resource list has changed
type ResourceListChangedNotification struct {
	Method string `json:"method"`
}

// SubscribeRequest represents a request to subscribe to resource changes
type SubscribeRequest struct {
	Method string `json:"method"`
	URI    string `json:"uri"`
}

// UnsubscribeRequest represents a request to unsubscribe from resource changes
type UnsubscribeRequest struct {
	Method string `json:"method"`
	URI    string `json:"uri"`
}

// ResourceUpdatedNotification represents a notification that a resource has been updated
type ResourceUpdatedNotification struct {
	Method string `json:"method"`
	URI    string `json:"uri"`
}
