package client

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

func setupTestSamplingClient(t *testing.T) (*SamplingClient, *mock.MockTransport, context.Context, context.CancelFunc) {
	logger := testutil.NewTestLogger(t)
	mockTransport := mock.NewMockTransport(logger)
	baseClient := base.NewClient(mockTransport)
	client := NewSamplingClient(baseClient)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// Start the client and transport
	err := mockTransport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}

	err = baseClient.Start(ctx)
	if err != nil {
		cancel()
		t.Fatalf("Failed to start client: %v", err)
	}

	return client, mockTransport, ctx, cancel
}

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
			client, mockTransport, ctx, cancel := setupTestSamplingClient(t)
			defer cancel()

			mockTransport.ClearSentMessages()
			done := make(chan struct{})

			go func() {
				defer close(done)
				select {
				case msg := <-mockTransport.GetRouter().Requests:
					if msg.Method != methods.SampleCreate {
						t.Errorf("Expected method %s, got %s", methods.SampleCreate, msg.Method)
					}

					// Verify request parameters
					var req types.CreateMessageRequest
					if err := json.Unmarshal(*msg.Params, &req); err != nil {
						t.Errorf("Failed to unmarshal request params: %v", err)
						return
					}

					response := &types.Message{
						JSONRPC: types.JSONRPCVersion,
						ID:      msg.ID,
					}

					if tt.wantErr {
						response.Error = &types.ErrorResponse{
							Code:    tt.errCode,
							Message: tt.errMsg,
						}
					} else {
						data, err := testutil.MarshalResult(tt.want)
						if err != nil {
							t.Errorf("Failed to marshal result: %v", err)
							return
						}
						response.Result = data
					}

					mockTransport.SimulateReceive(ctx, response)

				case <-ctx.Done():
					t.Error("Context cancelled while waiting for request")
					return
				}
			}()

			result, err := client.CreateMessage(ctx, tt.req)
			<-done

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
		})
	}
}

func TestSamplingClient_CreateMessageWithDefaults(t *testing.T) {
	client, mockTransport, ctx, cancel := setupTestSamplingClient(t)
	defer cancel()

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

	done := make(chan struct{})

	go func() {
		defer close(done)
		select {
		case msg := <-mockTransport.GetRouter().Requests:
			if msg.Method != methods.SampleCreate {
				t.Errorf("Expected method %s, got %s", methods.SampleCreate, msg.Method)
			}

			// Verify request uses default values
			var req types.CreateMessageRequest
			if err := json.Unmarshal(*msg.Params, &req); err != nil {
				t.Errorf("Failed to unmarshal request params: %v", err)
				return
			}

			if req.MaxTokens != 1000 {
				t.Errorf("Expected default MaxTokens 1000, got %d", req.MaxTokens)
			}

			response := &types.Message{
				JSONRPC: types.JSONRPCVersion,
				ID:      msg.ID,
			}

			data, err := testutil.MarshalResult(want)
			if err != nil {
				t.Errorf("Failed to marshal result: %v", err)
				return
			}
			response.Result = data

			mockTransport.SimulateReceive(ctx, response)

		case <-ctx.Done():
			t.Error("Context cancelled while waiting for request")
			return
		}
	}()

	result, err := client.CreateMessageWithDefaults(ctx, messages)
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
}
