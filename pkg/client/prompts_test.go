package client

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dwrtz/mcp-go/internal/base"
	"github.com/dwrtz/mcp-go/internal/testutil"
	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

func setupTestPromptsClient(t *testing.T) (*PromptsClient, *transport.MockTransport, context.Context, context.CancelFunc) {
	logger := testutil.NewTestLogger(t)
	mockTransport := transport.NewMockTransport(logger)
	baseClient := base.NewClient(mockTransport)
	client := NewPromptsClient(baseClient)

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

func TestPromptsClient_List(t *testing.T) {
	tests := []struct {
		name     string
		prompts  []types.Prompt
		wantErr  bool
		errorMsg string
	}{
		{
			name: "successful prompt listing",
			prompts: []types.Prompt{
				{
					Name:        "code_review",
					Description: "Review code for quality and improvements",
					Arguments: []types.PromptArgument{
						{
							Name:        "code",
							Description: "The code to review",
							Required:    true,
						},
					},
				},
				{
					Name:        "summarize",
					Description: "Summarize text content",
					Arguments: []types.PromptArgument{
						{
							Name:     "text",
							Required: true,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "empty prompt list",
			prompts: []types.Prompt{},
			wantErr: false,
		},
		{
			name:     "server error",
			wantErr:  true,
			errorMsg: "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockTransport, ctx, cancel := setupTestPromptsClient(t)
			defer cancel()

			mockTransport.ClearSentMessages()

			// Channel to coordinate test completion
			done := make(chan struct{})

			// Handle mock responses
			go func() {
				defer close(done)
				select {
				case msg := <-mockTransport.GetRouter().Requests:
					if msg.Method != methods.ListPrompts {
						t.Errorf("Expected method %s, got %s", methods.ListPrompts, msg.Method)
					}

					response := &types.Message{
						JSONRPC: types.JSONRPCVersion,
						ID:      msg.ID,
					}

					if tt.wantErr {
						response.Error = &types.ErrorResponse{
							Code:    types.InternalError,
							Message: tt.errorMsg,
						}
					} else {
						result := &types.ListPromptsResult{
							Prompts: tt.prompts,
						}
						data, err := testutil.MarshalResult(result)
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

			// Make the request
			prompts, err := client.List(ctx)

			// Wait for mock handler to complete
			<-done

			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(prompts) != len(tt.prompts) {
					t.Errorf("Expected %d prompts, got %d", len(tt.prompts), len(prompts))
				}

				for i, want := range tt.prompts {
					if i >= len(prompts) {
						t.Errorf("Missing prompt at index %d", i)
						continue
					}
					if prompts[i].Name != want.Name {
						t.Errorf("Prompt %d Name mismatch: want %s, got %s", i, want.Name, prompts[i].Name)
					}
					if prompts[i].Description != want.Description {
						t.Errorf("Prompt %d Description mismatch: want %s, got %s", i, want.Description, prompts[i].Description)
					}
				}
			}
		})
	}
}

func TestPromptsClient_Get(t *testing.T) {
	tests := []struct {
		name       string
		promptName string
		args       map[string]string
		want       *types.GetPromptResult
		wantErr    bool
		errorMsg   string
	}{
		{
			name:       "successful prompt retrieval",
			promptName: "code_review",
			args: map[string]string{
				"code": "def hello():\n    print('world')",
			},
			want: &types.GetPromptResult{
				Description: "Code review prompt",
				Messages: []types.PromptMessage{
					{
						Role: types.RoleUser,
						Content: types.TextContent{
							Type: "text",
							Text: "Please review this Python code:\ndef hello():\n    print('world')",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "prompt not found",
			promptName: "nonexistent",
			wantErr:    true,
			errorMsg:   "prompt not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockTransport, ctx, cancel := setupTestPromptsClient(t)
			defer cancel()

			mockTransport.ClearSentMessages()
			done := make(chan struct{})

			go func() {
				defer close(done)
				select {
				case msg := <-mockTransport.GetRouter().Requests:
					if msg.Method != methods.GetPrompt {
						t.Errorf("Expected method %s, got %s", methods.GetPrompt, msg.Method)
					}

					var req types.GetPromptRequest
					if err := json.Unmarshal(*msg.Params, &req); err != nil {
						t.Errorf("Failed to unmarshal request params: %v", err)
						return
					}

					if req.Name != tt.promptName {
						t.Errorf("Expected prompt name %s, got %s", tt.promptName, req.Name)
					}

					response := &types.Message{
						JSONRPC: types.JSONRPCVersion,
						ID:      msg.ID,
					}

					if tt.wantErr {
						response.Error = &types.ErrorResponse{
							Code:    types.InvalidParams,
							Message: tt.errorMsg,
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

			got, err := client.Get(ctx, tt.promptName, tt.args)
			<-done

			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.Description != tt.want.Description {
					t.Errorf("Description mismatch: want %s, got %s", tt.want.Description, got.Description)
				}

				if len(got.Messages) != len(tt.want.Messages) {
					t.Errorf("Expected %d messages, got %d", len(tt.want.Messages), len(got.Messages))
				}

				for i, wantMsg := range tt.want.Messages {
					if i >= len(got.Messages) {
						t.Errorf("Missing message at index %d", i)
						continue
					}
					if got.Messages[i].Role != wantMsg.Role {
						t.Errorf("Message %d Role mismatch: want %s, got %s", i, wantMsg.Role, got.Messages[i].Role)
					}
					// Compare content - this assumes TextContent, but could be extended for other types
					wantContent, ok := wantMsg.Content.(types.TextContent)
					if !ok {
						t.Errorf("Message %d: expected TextContent", i)
						continue
					}
					gotContent, ok := got.Messages[i].Content.(types.TextContent)
					if !ok {
						t.Errorf("Message %d: got unexpected content type", i)
						continue
					}
					if gotContent.Text != wantContent.Text {
						t.Errorf("Message %d Text mismatch: want %s, got %s", i, wantContent.Text, gotContent.Text)
					}
				}
			}
		})
	}
}

func TestPromptsClient_OnPromptListChanged(t *testing.T) {
	client, mockTransport, ctx, cancel := setupTestPromptsClient(t)
	defer cancel()

	// Channel to track callback invocation
	callbackInvoked := make(chan struct{})

	// Register callback
	client.OnPromptListChanged(func() {
		close(callbackInvoked)
	})

	// Simulate server sending a notification
	notification := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		Method:  methods.PromptsChanged,
		Params:  &json.RawMessage{'{', '}'}, // Empty JSON object
	}
	mockTransport.SimulateReceive(ctx, notification)

	// Wait for callback with timeout
	select {
	case <-callbackInvoked:
		// Success - callback was invoked
	case <-time.After(time.Second):
		t.Error("Timeout waiting for prompt list changed callback")
	case <-ctx.Done():
		t.Error("Context cancelled while waiting for callback")
	}
}
