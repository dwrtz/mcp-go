package sampling

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/internal/mock"
	"github.com/dwrtz/mcp-go/internal/testutil"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

func setupTest(t *testing.T) (context.Context, *base.Base, *Client, func()) {
	logger := testutil.NewTestLogger(t)
	serverTransport, clientTransport := mock.NewMockPipeTransports(logger)

	baseServer := base.NewBase(serverTransport)
	baseClient := base.NewBase(clientTransport)

	handler := func(ctx context.Context, req *types.CreateMessageRequest) (*types.CreateMessageResult, error) {
		// Validate request
		if len(req.Messages) == 0 {
			return nil, types.NewError(types.InvalidParams, "messages array cannot be empty")
		}
		if req.MaxTokens <= 0 {
			return nil, types.NewError(types.InvalidParams, "maxTokens must be positive")
		}

		return &types.CreateMessageResult{
			Role: types.RoleAssistant,
			Content: types.TextContent{
				Type: "text",
				Text: "Sample response",
			},
			Model:      "sample-model",
			StopReason: "endTurn",
		}, nil
	}

	samplingClient := NewClient(baseClient, handler)

	ctx := context.Background()
	if err := baseServer.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	if err := baseClient.Start(ctx); err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}

	cleanup := func() {
		baseClient.Close()
		baseServer.Close()
	}

	return ctx, baseServer, samplingClient, cleanup
}

func TestClient_HandleCreateMessageRequest(t *testing.T) {
	tests := []struct {
		name          string
		request       *types.CreateMessageRequest
		wantErr       bool
		expectedModel string
	}{
		{
			name: "valid request",
			request: &types.CreateMessageRequest{
				Messages: []types.SamplingMessage{
					{
						Role: types.RoleUser,
						Content: types.TextContent{
							Type: "text",
							Text: "Hello!",
						},
					},
				},
				MaxTokens: 100,
				ModelPreferences: &types.ModelPreferences{
					Hints: []types.ModelHint{
						{Name: "claude-3-sonnet"},
					},
					IntelligencePriority: 0.8,
					SpeedPriority:        0.5,
				},
			},
			wantErr:       false,
			expectedModel: "sample-model", // matches what's returned in handleCreateMessage
		},
		{
			name: "empty messages",
			request: &types.CreateMessageRequest{
				Messages:  []types.SamplingMessage{},
				MaxTokens: 100,
			},
			wantErr: true,
		},
		{
			name: "invalid max tokens",
			request: &types.CreateMessageRequest{
				Messages: []types.SamplingMessage{
					{
						Role: types.RoleUser,
						Content: types.TextContent{
							Type: "text",
							Text: "Hello!",
						},
					},
				},
				MaxTokens: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, baseServer, _, cleanup := setupTest(t)
			defer cleanup()

			// Send request from server to client
			resp, err := baseServer.SendRequest(ctx, methods.SampleCreate, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Parse response
			var result types.CreateMessageResult
			if err := json.Unmarshal(*resp.Result, &result); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			// Verify response fields
			if result.Role != types.RoleAssistant {
				t.Errorf("Expected role %q, got %q", types.RoleAssistant, result.Role)
			}

			content, ok := result.Content.(types.TextContent)
			if !ok {
				t.Errorf("Expected TextContent, got %T", result.Content)
				return
			}

			if content.Type != "text" {
				t.Errorf("Expected content type %q, got %q", "text", content.Type)
			}

			if result.Model != tt.expectedModel {
				t.Errorf("Expected model %q, got %q", tt.expectedModel, result.Model)
			}

			if result.StopReason != "endTurn" {
				t.Errorf("Expected stopReason %q, got %q", "endTurn", result.StopReason)
			}
		})
	}
}

func TestClient_HandleCreateMessageRequest_WithContext(t *testing.T) {
	ctx, baseServer, _, cleanup := setupTest(t)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(ctx)

	// Prepare request
	req := &types.CreateMessageRequest{
		Messages: []types.SamplingMessage{
			{
				Role: types.RoleUser,
				Content: types.TextContent{
					Type: "text",
					Text: "Hello!",
				},
			},
		},
		MaxTokens: 100,
	}

	// Send the request in a goroutine
	errChan := make(chan error, 1)
	go func() {
		_, err := baseServer.SendRequest(ctx, methods.SampleCreate, req)
		errChan <- err
	}()

	// Immediately cancel
	cancel()

	// Wait for the request call to fail
	select {
	case err := <-errChan:
		if err == nil {
			t.Error("Expected error after context cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for cancelled request")
	}

	cleanup()
}
