package sampling

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/internal/mock"
	"github.com/dwrtz/mcp-go/internal/testutil"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

func setupTest(t *testing.T) (context.Context, *SamplingServer, *base.Base, func()) {
	logger := testutil.NewTestLogger(t)
	serverTransport, clientTransport := mock.NewMockPipeTransports(logger)

	baseServer := base.NewBase(serverTransport)
	baseClient := base.NewBase(clientTransport)
	samplingServer := NewSamplingServer(baseServer)

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

	return ctx, samplingServer, baseClient, cleanup
}

// mockSamplingHandler provides a basic mock implementation for testing
func mockSamplingHandler(_ context.Context, req *types.CreateMessageRequest) (*types.CreateMessageResult, error) {
	// Validate basic requirements
	if len(req.Messages) == 0 {
		return nil, types.NewError(types.InvalidParams, "messages array cannot be empty")
	}
	if req.MaxTokens <= 0 {
		return nil, types.NewError(types.InvalidParams, "maxTokens must be positive")
	}

	// Create a standard response
	return &types.CreateMessageResult{
		Role: types.RoleAssistant,
		Content: types.TextContent{
			Type: "text",
			Text: "This is a mock response",
		},
		Model:      "mock-model",
		StopReason: "endTurn",
	}, nil
}

func TestSamplingServer_CreateMessage(t *testing.T) {
	tests := []struct {
		name      string
		messages  []types.SamplingMessage
		modelPref *types.ModelPreferences
		maxTokens int
		wantErr   bool
		errMsg    string
	}{
		{
			name: "successful message creation",
			messages: []types.SamplingMessage{
				{
					Role: types.RoleUser,
					Content: types.TextContent{
						Type: "text",
						Text: "Hello, how are you?",
					},
				},
			},
			modelPref: &types.ModelPreferences{
				Hints: []types.ModelHint{
					{Name: "claude-3-sonnet"},
				},
				IntelligencePriority: 0.8,
				SpeedPriority:        0.5,
			},
			maxTokens: 100,
			wantErr:   false,
		},
		{
			name:      "empty messages array",
			messages:  []types.SamplingMessage{},
			maxTokens: 100,
			wantErr:   true,
			errMsg:    "messages array cannot be empty",
		},
		{
			name: "invalid max tokens",
			messages: []types.SamplingMessage{
				{
					Role: types.RoleUser,
					Content: types.TextContent{
						Type: "text",
						Text: "Hello",
					},
				},
			},
			maxTokens: 0,
			wantErr:   true,
			errMsg:    "maxTokens must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, server, client, cleanup := setupTest(t)
			defer cleanup()

			// Register mock handler
			client.RegisterRequestHandler(methods.SampleCreate, func(ctx context.Context, params *json.RawMessage) (interface{}, error) {
				if params == nil {
					return nil, types.NewError(types.InvalidParams, "missing params")
				}
				var req types.CreateMessageRequest
				if err := json.Unmarshal(*params, &req); err != nil {
					return nil, err
				}
				return mockSamplingHandler(ctx, &req)
			})

			// Create request
			req := &types.CreateMessageRequest{
				Messages:         tt.messages,
				ModelPreferences: tt.modelPref,
				MaxTokens:        tt.maxTokens,
			}

			// Call CreateMessage
			result, err := server.CreateMessage(ctx, req)

			// Check for expected errors
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if mcpErr, ok := err.(*types.ErrorResponse); ok {
					if mcpErr.Message != tt.errMsg {
						t.Errorf("Expected error message %q, got %q", tt.errMsg, mcpErr.Message)
					}
				} else {
					t.Errorf("Expected MCP error, got %T", err)
				}
				return
			}

			// For successful cases, verify the response
			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			// Verify basic structure
			if result.Role != types.RoleAssistant {
				t.Errorf("Expected role %v, got %v", types.RoleAssistant, result.Role)
			}

			if _, ok := result.Content.(types.TextContent); !ok {
				t.Errorf("Expected TextContent, got %T", result.Content)
			}

			if result.Model == "" {
				t.Error("Model name should not be empty")
			}

			if result.StopReason == "" {
				t.Error("StopReason should not be empty")
			}
		})
	}
}
