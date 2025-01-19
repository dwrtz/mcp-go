package client

import (
	"testing"

	"github.com/dwrtz/mcp-go/internal/mock"
	"github.com/dwrtz/mcp-go/pkg/methods"
	"github.com/dwrtz/mcp-go/pkg/types"
)

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
			},
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
			// Create mock client with all dependencies
			mockClient := mock.NewMockClient(t)
			defer mockClient.Close()

			// Create prompts client
			client := NewPromptsClient(mockClient.BaseClient)

			// Handle the expected request
			done := mockClient.ExpectRequest(methods.ListPrompts, func(msg *types.Message) *types.Message {
				if tt.wantErr {
					return mockClient.CreateErrorResponse(msg, types.InternalError, tt.errorMsg, nil)
				}
				return mockClient.CreateSuccessResponse(msg, &types.ListPromptsResult{
					Prompts: tt.prompts,
				})
			})

			// Make the actual request
			prompts, err := client.List(mockClient.Context)

			// Wait for mock handler to complete
			<-done

			// Verify results
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

			mockClient.AssertNoErrors(t)
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
			mockClient := mock.NewMockClient(t)
			defer mockClient.Close()

			client := NewPromptsClient(mockClient.BaseClient)

			done := mockClient.ExpectRequest(methods.GetPrompt, func(msg *types.Message) *types.Message {
				if tt.wantErr {
					return mockClient.CreateErrorResponse(msg, types.InvalidParams, tt.errorMsg, nil)
				}
				return mockClient.CreateSuccessResponse(msg, tt.want)
			})

			got, err := client.Get(mockClient.Context, tt.promptName, tt.args)
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
					// Compare content
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

			mockClient.AssertNoErrors(t)
		})
	}
}

func TestPromptsClient_OnPromptListChanged(t *testing.T) {
	mockClient := mock.NewMockClient(t)
	defer mockClient.Close()

	client := NewPromptsClient(mockClient.BaseClient)

	// Channel to track callback invocation
	callbackInvoked := make(chan struct{})

	// Register callback
	client.OnPromptListChanged(func() {
		close(callbackInvoked)
	})

	// Test notification handling
	err := mockClient.SimulateNotification(methods.PromptsChanged, struct{}{})
	if err != nil {
		t.Fatalf("Failed to simulate notification: %v", err)
	}

	// Wait for callback with timeout
	if err := mockClient.WaitForCallback(func(done chan<- struct{}) {
		select {
		case <-callbackInvoked:
			close(done)
		case <-mockClient.Context.Done():
			t.Error("Context cancelled while waiting for callback")
		}
	}); err != nil {
		t.Errorf("Error waiting for callback: %v", err)
	}

	mockClient.AssertNoErrors(t)
}
