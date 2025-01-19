package prompts

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

func setupTest(t *testing.T) (context.Context, *PromptsClient, *base.Server, func()) {
	logger := testutil.NewTestLogger(t)
	serverTransport, clientTransport := mock.NewMockPipeTransports(logger)
	baseServer := base.NewServer(serverTransport)
	baseClient := base.NewClient(clientTransport)

	promptsClient := NewPromptsClient(baseClient)

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

	return ctx, promptsClient, baseServer, cleanup
}

func TestPromptsClient_List(t *testing.T) {
	testCases := []struct {
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
		},
		{
			name:     "server error",
			wantErr:  true,
			errorMsg: "internal server error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, client, server, cleanup := setupTest(t)
			defer cleanup()

			server.RegisterRequestHandler(methods.ListPrompts, func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				if tc.wantErr {
					return nil, types.NewError(types.InternalError, tc.errorMsg)
				}
				return &types.ListPromptsResult{Prompts: tc.prompts}, nil
			})

			prompts, err := client.List(ctx)

			if (err != nil) != tc.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				if len(prompts) != len(tc.prompts) {
					t.Errorf("Expected %d prompts, got %d", len(tc.prompts), len(prompts))
					return
				}

				for i, want := range tc.prompts {
					got := prompts[i]
					if got.Name != want.Name {
						t.Errorf("Prompt %d name = %v, want %v", i, got.Name, want.Name)
					}
					if got.Description != want.Description {
						t.Errorf("Prompt %d description = %v, want %v", i, got.Description, want.Description)
					}
				}
			}
		})
	}
}

func TestPromptsClient_Get(t *testing.T) {
	testCases := []struct {
		name       string
		promptName string
		args       map[string]string
		want       *types.GetPromptResult
		wantErr    bool
		errorCode  int
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
		},
		{
			name:       "prompt not found",
			promptName: "nonexistent",
			wantErr:    true,
			errorCode:  types.InvalidParams,
			errorMsg:   "prompt not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, client, server, cleanup := setupTest(t)
			defer cleanup()

			server.RegisterRequestHandler(methods.GetPrompt, func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				if tc.wantErr {
					return nil, types.NewError(tc.errorCode, tc.errorMsg)
				}
				return tc.want, nil
			})

			result, err := client.Get(ctx, tc.promptName, tc.args)

			if (err != nil) != tc.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				if result.Description != tc.want.Description {
					t.Errorf("Description = %v, want %v", result.Description, tc.want.Description)
				}

				if len(result.Messages) != len(tc.want.Messages) {
					t.Errorf("Got %d messages, want %d", len(result.Messages), len(tc.want.Messages))
					return
				}

				for i, wantMsg := range tc.want.Messages {
					gotMsg := result.Messages[i]
					if gotMsg.Role != wantMsg.Role {
						t.Errorf("Message[%d].Role = %v, want %v", i, gotMsg.Role, wantMsg.Role)
					}

					wantContent, ok := wantMsg.Content.(types.TextContent)
					if !ok {
						t.Errorf("Message[%d]: expected TextContent", i)
						continue
					}

					gotContent, ok := gotMsg.Content.(types.TextContent)
					if !ok {
						t.Errorf("Message[%d]: got unexpected content type", i)
						continue
					}

					if gotContent.Text != wantContent.Text {
						t.Errorf("Message[%d].Text = %v, want %v", i, gotContent.Text, wantContent.Text)
					}
				}
			}
		})
	}
}

func TestPromptsClient_OnPromptListChanged(t *testing.T) {
	ctx, client, server, cleanup := setupTest(t)
	defer cleanup()

	callbackInvoked := make(chan struct{})

	client.OnPromptListChanged(func() {
		close(callbackInvoked)
	})

	notification := struct{}{}
	if err := server.SendNotification(ctx, methods.PromptsChanged, notification); err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	select {
	case <-callbackInvoked:
		// Success
	case <-time.After(time.Second):
		t.Error("Callback not called within timeout")
	}
}
