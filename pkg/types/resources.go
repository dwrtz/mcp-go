package types

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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

func (TextResourceContents) isResourceContent() {}

// BlobResourceContents represents binary resource contents
type BlobResourceContents struct {
	ResourceContents
	Blob string `json:"blob"` // base64-encoded data
}

func (BlobResourceContents) isResourceContent() {}

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

// ResourceContent is an interface each content struct implements.
type ResourceContent interface {
	// Just a sentinel method so these types can be recognized as resource contents.
	isResourceContent()
}

// ReadResourceResult represents the response to a resources/read request
type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"` // Can be TextResourceContents or BlobResourceContents
}

// UnmarshalJSON implements json.Unmarshaler for ReadResourceResult
func (r *ReadResourceResult) UnmarshalJSON(data []byte) error {
	// We'll parse into a temp struct that has `Contents` as raw JSON.
	type alias ReadResourceResult
	tmp := &struct {
		Contents []json.RawMessage `json:"contents"`
		*alias
	}{
		alias: (*alias)(r),
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	// We'll clear out r.Contents so we can rebuild it below.
	r.Contents = make([]ResourceContent, 0, len(tmp.Contents))

	for _, raw := range tmp.Contents {
		// Quick approach: decode into a map and see if "blob" or "text" is present.
		var objMap map[string]interface{}
		if err := json.Unmarshal(raw, &objMap); err != nil {
			return err
		}

		switch {
		// If there's a "blob" key, treat it as BlobResourceContents
		case objMap["blob"] != nil:
			var blobC BlobResourceContents
			if err := json.Unmarshal(raw, &blobC); err != nil {
				return err
			}
			r.Contents = append(r.Contents, blobC)

		// If there's a "text" key, treat it as TextResourceContents
		case objMap["text"] != nil:
			var textC TextResourceContents
			if err := json.Unmarshal(raw, &textC); err != nil {
				return err
			}
			r.Contents = append(r.Contents, textC)

		default:
			return fmt.Errorf("couldn't guess resource type: neither 'blob' nor 'text' found")
		}
	}

	return nil
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
