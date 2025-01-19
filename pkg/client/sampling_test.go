package client

import (
	"encoding/json"
	"testing"

	"github.com/dwrtz/mcp-go/internal/mock"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

func TestSamplingClient_CreateMessage(t *testing.T) {
	tests := []struct {
		name    string
		req     *types.CreateMessageRequest
		want    *types.CreateMessageResult
		wantErr bool
		errCode int
		errMsg  string
	}{
		{
			name: "successful text completion",
			req: &types.CreateMessageRequest{
				Method: methods.SampleCreate,
				Messages: []types.SamplingMessage{
					{
						Role: types.RoleUser,
						Content: types.TextContent{
							Type: "text",
							Text: "What is the capital of France?",
						},
					},
				},
				MaxTokens: 100,
				ModelPreferences: &types.ModelPreferences{
					IntelligencePriority: 0.8,
					SpeedPriority:        0.5,
				},
			},
			want: &types.CreateMessageResult{
				Role: types.RoleAssistant,
				Content: types.TextContent{
					Type: "text",
					Text: "The capital of France is Paris.",
				},
				Model:      "claude-3-sonnet-20240307",
				StopReason: "endTurn",
			},
			wantErr: false,
		},
		{
			name: "request with invalid tokens",
			req: &types.CreateMessageRequest{
				Method: methods.SampleCreate,
				Messages: []types.SamplingMessage{
					{
						Role: types.RoleUser,
						Content: types.TextContent{
							Type: "text",
							Text: "Test message",
						},
					},
				},
				MaxTokens: -1, // Invalid
			},
			wantErr: true,
			errCode: types.InvalidParams,
			errMsg:  "maxTokens must be positive",
		},
		{
			name: "request with empty messages",
			req: &types.CreateMessageRequest{
				Method:    methods.SampleCreate,
				Messages:  []types.SamplingMessage{},
				MaxTokens: 100,
			},
			wantErr: true,
			errCode: types.InvalidParams,
			errMsg:  "messages array cannot be empty",
		},
		{
			name: "successful message with model hints",
			req: &types.CreateMessageRequest{
				Method: methods.SampleCreate,
				Messages: []types.SamplingMessage{
					{
						Role: types.RoleUser,
						Content: types.TextContent{
							Type: "text",
							Text: "Hello",
						},
					},
				},
				ModelPreferences: &types.ModelPreferences{
					Hints: []types.ModelHint{
						{Name: "claude-3-sonnet"},
					},
				},
				MaxTokens: 100,
			},
			want: &types.CreateMessageResult{
				Role: types.RoleAssistant,
				Content: types.TextContent{
					Type: "text",
					Text: "Hi there! How can I help you today?",
				},
				Model:      "claude-3-sonnet-20240307",
				StopReason: "endTurn",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client with all dependencies
			mockClient := mock.NewMockClient(t)
			defer mockClient.Close()

			// Create sampling client
			client := NewSamplingClient(mockClient.BaseClient)

			// Handle the expected request
			done := mockClient.ExpectRequest(methods.SampleCreate, func(msg *types.Message) *types.Message {
				if tt.wantErr {
					return mockClient.CreateErrorResponse(msg, tt.errCode, tt.errMsg, nil)
				}
				return mockClient.CreateSuccessResponse(msg, tt.want)
			})

			// Make the actual request
			result, err := client.CreateMessage(mockClient.Context, tt.req)

			// Wait for mock handler to complete
			<-done

			// Verify results
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if mcpErr, ok := err.(*types.ErrorResponse); ok {
					if mcpErr.Code != tt.errCode {
						t.Errorf("Expected error code %d, got %d", tt.errCode, mcpErr.Code)
					}
					if mcpErr.Message != tt.errMsg {
						t.Errorf("Expected error message %s, got %s", tt.errMsg, mcpErr.Message)
					}
				} else {
					t.Errorf("Expected MCP error, got %T", err)
				}
			} else {
				if result.Role != tt.want.Role {
					t.Errorf("Role mismatch: want %s, got %s", tt.want.Role, result.Role)
				}
				if result.Model != tt.want.Model {
					t.Errorf("Model mismatch: want %s, got %s", tt.want.Model, result.Model)
				}
				if result.StopReason != tt.want.StopReason {
					t.Errorf("StopReason mismatch: want %s, got %s", tt.want.StopReason, result.StopReason)
				}

				// Compare content
				wantContent, ok := tt.want.Content.(types.TextContent)
				if !ok {
					t.Errorf("Expected TextContent in want")
					return
				}
				gotContent, ok := result.Content.(types.TextContent)
				if !ok {
					t.Errorf("Expected TextContent in result")
					return
				}
				if gotContent.Text != wantContent.Text {
					t.Errorf("Content mismatch:\nwant: %s\ngot:  %s", wantContent.Text, gotContent.Text)
				}
			}

			mockClient.AssertNoErrors(t)
		})
	}
}

func TestSamplingClient_CreateMessageWithDefaults(t *testing.T) {
	mockClient := mock.NewMockClient(t)
	defer mockClient.Close()

	client := NewSamplingClient(mockClient.BaseClient)

	messages := []types.SamplingMessage{
		{
			Role: types.RoleUser,
			Content: types.TextContent{
				Type: "text",
				Text: "Hello",
			},
		},
	}

	want := &types.CreateMessageResult{
		Role: types.RoleAssistant,
		Content: types.TextContent{
			Type: "text",
			Text: "Hello! How can I assist you today?",
		},
		Model:      "claude-3-sonnet-20240307",
		StopReason: "endTurn",
	}

	done := mockClient.ExpectRequest(methods.SampleCreate, func(msg *types.Message) *types.Message {
		// Verify request uses default values
		var req types.CreateMessageRequest
		if err := json.Unmarshal(*msg.Params, &req); err != nil {
			t.Errorf("Failed to unmarshal request params: %v", err)
			return nil
		}

		if req.MaxTokens != 1000 {
			t.Errorf("Expected default MaxTokens 1000, got %d", req.MaxTokens)
		}

		return mockClient.CreateSuccessResponse(msg, want)
	})

	result, err := client.CreateMessageWithDefaults(mockClient.Context, messages)
	<-done

	if err != nil {
		t.Fatalf("CreateMessageWithDefaults() error = %v", err)
	}

	// Verify response
	if result.Role != want.Role {
		t.Errorf("Role mismatch: want %s, got %s", want.Role, result.Role)
	}

	gotContent, ok := result.Content.(types.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent in result")
	}
	wantContent := want.Content.(types.TextContent)
	if gotContent.Text != wantContent.Text {
		t.Errorf("Content mismatch:\nwant: %s\ngot:  %s", wantContent.Text, gotContent.Text)
	}

	mockClient.AssertNoErrors(t)
}
