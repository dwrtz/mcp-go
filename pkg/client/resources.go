package client

import (
	"context"
	"encoding/json"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// ResourcesClient provides client-side resource functionality
type ResourcesClient struct {
	base *base.Client
}

// NewResourcesClient creates a new ResourcesClient
func NewResourcesClient(base *base.Client) *ResourcesClient {
	return &ResourcesClient{base: base}
}

// List requests the list of available resources
func (c *ResourcesClient) List(ctx context.Context) ([]types.Resource, error) {
	req := &types.ListResourcesRequest{
		Method: methods.ListResources,
	}

	resp, err := c.base.SendRequest(ctx, methods.ListResources, req)
	if err != nil {
		return nil, err
	}

	var result types.ListResourcesResult
	if err := json.Unmarshal(*resp.Result, &result); err != nil {
		return nil, err
	}

	return result.Resources, nil
}

// Read requests the contents of a specific resource
func (c *ResourcesClient) Read(ctx context.Context, uri string) ([]interface{}, error) {
	req := &types.ReadResourceRequest{
		Method: methods.ReadResource,
		URI:    uri,
	}

	resp, err := c.base.SendRequest(ctx, methods.ReadResource, req)
	if err != nil {
		return nil, err
	}

	var result types.ReadResourceResult
	if err := json.Unmarshal(*resp.Result, &result); err != nil {
		return nil, err
	}

	return result.Contents, nil
}

// ListTemplates requests the list of available resource templates
func (c *ResourcesClient) ListTemplates(ctx context.Context) ([]types.ResourceTemplate, error) {
	req := &types.ListResourceTemplatesRequest{
		Method: methods.ListResourceTemplates,
	}

	resp, err := c.base.SendRequest(ctx, methods.ListResourceTemplates, req)
	if err != nil {
		return nil, err
	}

	var result types.ListResourceTemplatesResult
	if err := json.Unmarshal(*resp.Result, &result); err != nil {
		return nil, err
	}

	return result.ResourceTemplates, nil
}

// Subscribe subscribes to updates for a specific resource
func (c *ResourcesClient) Subscribe(ctx context.Context, uri string) error {
	req := &types.SubscribeRequest{
		Method: methods.SubscribeResource,
		URI:    uri,
	}

	_, err := c.base.SendRequest(ctx, methods.SubscribeResource, req)
	return err
}

// Unsubscribe unsubscribes from updates for a specific resource
func (c *ResourcesClient) Unsubscribe(ctx context.Context, uri string) error {
	req := &types.UnsubscribeRequest{
		Method: methods.UnsubscribeResource,
		URI:    uri,
	}

	_, err := c.base.SendRequest(ctx, methods.UnsubscribeResource, req)
	return err
}

// OnResourceUpdated registers a callback for resource update notifications
func (c *ResourcesClient) OnResourceUpdated(callback func(uri string)) {
	c.base.RegisterNotificationHandler(methods.ResourceUpdated, func(ctx context.Context, params json.RawMessage) {
		var notif types.ResourceUpdatedNotification
		if err := json.Unmarshal(params, &notif); err != nil {
			c.base.Logf("Failed to parse resource updated notification: %v", err)
			return
		}
		callback(notif.URI)
	})
}

// OnResourceListChanged registers a callback for resource list change notifications
func (c *ResourcesClient) OnResourceListChanged(callback func()) {
	c.base.RegisterNotificationHandler(methods.ResourceListChanged, func(ctx context.Context, params json.RawMessage) {
		callback()
	})
}
