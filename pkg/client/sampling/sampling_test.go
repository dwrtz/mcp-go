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

func setupTest(t *testing.T) (context.Context, *base.Base, *SamplingClient, func()) {
	logger := testutil.NewTestLogger(t)
	serverTransport, clientTransport := mock.NewMockPipeTransports(logger)

	// Create server-side base and client that will make requests
	baseServer := base.NewBase(serverTransport)

	// Create client-side base and sampling client that will handle requests
	baseClient := base.NewBase(clientTransport)
	samplingClient := NewSamplingClient(baseClient)

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

func TestSamplingClient_HandleCreateMessageRequest(t *testing.T) {
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

func TestSamplingClient_HandleCreateMessageRequest_WithContext(t *testing.T) {
	ctx, baseServer, _, cleanup := setupTest(t)
	defer cleanup()

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

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

	// Send request in goroutine since we'll cancel it
	errChan := make(chan error)
	go func() {
		_, err := baseServer.SendRequest(ctx, methods.SampleCreate, req)
		errChan <- err
	}()

	// Cancel context and verify we get appropriate error
	cancel()

	select {
	case err := <-errChan:
		if err == nil {
			t.Error("Expected error after context cancellation")
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for cancelled request")
	}
}
