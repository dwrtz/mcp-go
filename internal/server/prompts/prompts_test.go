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

func setupTest(t *testing.T) (context.Context, *Server, *base.Base, func()) {
	logger := testutil.NewTestLogger(t)
	serverTransport, clientTransport := mock.NewMockPipeTransports(logger)
	baseServer := base.NewBase(serverTransport)
	baseClient := base.NewBase(clientTransport)

	initialPrompts := []types.Prompt{
		{
			Name:        "test_prompt",
			Description: "A test prompt",
			Arguments: []types.PromptArgument{
				{
					Name:        "arg1",
					Description: "First argument",
					Required:    true,
				},
			},
		},
	}
	promptsServer := NewServer(baseServer, initialPrompts)

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

	return ctx, promptsServer, baseClient, cleanup
}

func TestServer_ListPrompts(t *testing.T) {
	testCases := []struct {
		name    string
		prompts []types.Prompt
		wantErr bool
	}{
		{
			name: "successful prompts listing",
			prompts: []types.Prompt{
				{
					Name:        "test_prompt",
					Description: "A test prompt",
					Arguments: []types.PromptArgument{
						{
							Name:        "arg1",
							Description: "First argument",
							Required:    true,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "empty prompts list",
			prompts: []types.Prompt{},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, server, client, cleanup := setupTest(t)
			defer cleanup()

			// Set prompts on server
			if err := server.SetPrompts(ctx, tc.prompts); err != nil {
				t.Fatalf("Failed to set prompts: %v", err)
			}

			// Send list request
			resp, err := client.SendRequest(ctx, methods.ListPrompts, &types.ListPromptsRequest{
				Method: methods.ListPrompts,
			})

			if (err != nil) != tc.wantErr {
				t.Errorf("ListPrompts() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				var result types.ListPromptsResult
				if err := json.Unmarshal(*resp.Result, &result); err != nil {
					t.Fatalf("Failed to unmarshal result: %v", err)
				}

				if len(result.Prompts) != len(tc.prompts) {
					t.Errorf("Expected %d prompts, got %d", len(tc.prompts), len(result.Prompts))
					return
				}

				for i, want := range tc.prompts {
					got := result.Prompts[i]
					if got.Name != want.Name {
						t.Errorf("Prompt %d name = %v, want %v", i, got.Name, want.Name)
					}
					if got.Description != want.Description {
						t.Errorf("Prompt %d description = %v, want %v", i, got.Description, want.Description)
					}
					// Compare arguments if present
					if len(want.Arguments) > 0 {
						if len(got.Arguments) != len(want.Arguments) {
							t.Errorf("Prompt %d: expected %d arguments, got %d", i, len(want.Arguments), len(got.Arguments))
							continue
						}
						for j, wantArg := range want.Arguments {
							gotArg := got.Arguments[j]
							if gotArg.Name != wantArg.Name {
								t.Errorf("Prompt %d argument %d name = %v, want %v", i, j, gotArg.Name, wantArg.Name)
							}
							if gotArg.Description != wantArg.Description {
								t.Errorf("Prompt %d argument %d description = %v, want %v", i, j, gotArg.Description, wantArg.Description)
							}
							if gotArg.Required != wantArg.Required {
								t.Errorf("Prompt %d argument %d required = %v, want %v", i, j, gotArg.Required, wantArg.Required)
							}
						}
					}
				}
			}
		})
	}
}

func TestServer_GetPrompt(t *testing.T) {
	testCases := []struct {
		name       string
		promptName string
		args       map[string]string
		getter     func(ctx context.Context, args map[string]string) (*types.GetPromptResult, error)
		want       *types.GetPromptResult
		wantErr    bool
	}{
		{
			name:       "successful prompt retrieval",
			promptName: "test_prompt",
			args: map[string]string{
				"arg1": "value1",
			},
			getter: func(ctx context.Context, args map[string]string) (*types.GetPromptResult, error) {
				return &types.GetPromptResult{
					Description: "Test prompt result",
					Messages: []types.PromptMessage{
						{
							Role: types.RoleUser,
							Content: types.TextContent{
								Type: "text",
								Text: "Sample prompt with arg1=" + args["arg1"],
							},
						},
					},
				}, nil
			},
			want: &types.GetPromptResult{
				Description: "Test prompt result",
				Messages: []types.PromptMessage{
					{
						Role: types.RoleUser,
						Content: types.TextContent{
							Type: "text",
							Text: "Sample prompt with arg1=value1",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "prompt not found",
			promptName: "nonexistent",
			args:       map[string]string{},
			getter:     nil,
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, server, client, cleanup := setupTest(t)
			defer cleanup()

			if tc.getter != nil {
				server.RegisterPromptGetter(tc.promptName, tc.getter)
			}

			// Send get request
			resp, err := client.SendRequest(ctx, methods.GetPrompt, &types.GetPromptRequest{
				Method:    methods.GetPrompt,
				Name:      tc.promptName,
				Arguments: tc.args,
			})

			if (err != nil) != tc.wantErr {
				t.Errorf("GetPrompt() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr && resp.Result != nil {
				var result types.GetPromptResult
				if err := json.Unmarshal(*resp.Result, &result); err != nil {
					t.Fatalf("Failed to unmarshal result: %v", err)
				}

				// Compare messages
				if len(result.Messages) != len(tc.want.Messages) {
					t.Errorf("Expected %d messages, got %d", len(tc.want.Messages), len(result.Messages))
					return
				}

				for i, want := range tc.want.Messages {
					got := result.Messages[i]
					if got.Role != want.Role {
						t.Errorf("Message %d role = %v, want %v", i, got.Role, want.Role)
					}

					// Compare content
					gotContent, ok := got.Content.(types.TextContent)
					if !ok {
						t.Errorf("Message %d: expected TextContent", i)
						continue
					}

					wantContent, ok := want.Content.(types.TextContent)
					if !ok {
						t.Errorf("Message %d: want content is not TextContent", i)
						continue
					}

					if gotContent.Text != wantContent.Text {
						t.Errorf("Message %d text = %v, want %v", i, gotContent.Text, wantContent.Text)
					}
				}
			}
		})
	}
}

func TestServer_PromptsChanged(t *testing.T) {
	ctx, server, client, cleanup := setupTest(t)
	defer cleanup()

	// Channel to track notification reception
	notificationReceived := make(chan struct{})

	// Register notification handler on client
	client.RegisterNotificationHandler(methods.PromptsChanged, func(ctx context.Context, params json.RawMessage) {
		close(notificationReceived)
	})

	// Update prompts which should trigger notification
	prompts := []types.Prompt{
		{
			Name:        "new_prompt",
			Description: "A new prompt",
		},
	}

	if err := server.SetPrompts(ctx, prompts); err != nil {
		t.Fatalf("Failed to set prompts: %v", err)
	}

	// Wait for notification with timeout
	select {
	case <-notificationReceived:
		// Success
	case <-time.After(time.Second):
		t.Error("Timeout waiting for prompts changed notification")
	}
}
