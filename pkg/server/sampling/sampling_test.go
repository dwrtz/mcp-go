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

func setupTest(t *testing.T) (context.Context, *base.Client, *SamplingServer, func()) {
	logger := testutil.NewTestLogger(t)
	serverTransport, clientTransport := mock.NewMockPipeTransports(logger)
	baseServer := base.NewServer(serverTransport)
	baseClient := base.NewClient(clientTransport)

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

	return ctx, baseClient, samplingServer, cleanup
}

func TestSamplingServer_HandleCreateMessage(t *testing.T) {
	tests := []struct {
		name    string
		req     types.CreateMessageRequest
		want    *types.CreateMessageResult
		wantErr bool
		errCode int
		errMsg  string
	}{
		{
			name: "valid request",
			req: types.CreateMessageRequest{
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
				MaxTokens: 100,
				ModelPreferences: &types.ModelPreferences{
					IntelligencePriority: 0.8,
					SpeedPriority:        0.2,
				},
			},
			want: &types.CreateMessageResult{
				Role: types.RoleAssistant,
				Content: types.TextContent{
					Type: "text",
					Text: "Sample response",
				},
				Model:      "sample-model",
				StopReason: "endTurn",
			},
			wantErr: false,
		},
		{
			name: "empty messages array",
			req: types.CreateMessageRequest{
				Method:    methods.SampleCreate,
				Messages:  []types.SamplingMessage{},
				MaxTokens: 100,
			},
			wantErr: true,
			errCode: types.InvalidParams,
			errMsg:  "messages array cannot be empty",
		},
		{
			name: "invalid max tokens",
			req: types.CreateMessageRequest{
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
				MaxTokens: -1,
			},
			wantErr: true,
			errCode: types.InvalidParams,
			errMsg:  "maxTokens must be positive",
		},
		{
			name: "valid request with system prompt",
			req: types.CreateMessageRequest{
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
				MaxTokens:    100,
				SystemPrompt: "You are a helpful assistant.",
			},
			want: &types.CreateMessageResult{
				Role: types.RoleAssistant,
				Content: types.TextContent{
					Type: "text",
					Text: "Sample response",
				},
				Model:      "sample-model",
				StopReason: "endTurn",
			},
			wantErr: false,
		},
		{
			name: "valid request with stop sequences",
			req: types.CreateMessageRequest{
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
				MaxTokens:     100,
				StopSequences: []string{"END", "STOP"},
			},
			want: &types.CreateMessageResult{
				Role: types.RoleAssistant,
				Content: types.TextContent{
					Type: "text",
					Text: "Sample response",
				},
				Model:      "sample-model",
				StopReason: "endTurn",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, client, _, cleanup := setupTest(t)
			defer cleanup()

			// Send request
			resp, err := client.SendRequest(ctx, methods.SampleCreate, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err != nil {
					// The transport returned an actual error, so resp is nil. We can
					// check if it's an *types.ErrorResponse (or just check err.Error()).
					if mcpErr, ok := err.(*types.ErrorResponse); ok {
						if mcpErr.Code != tt.errCode {
							t.Errorf("Expected error code %d, got %d", tt.errCode, mcpErr.Code)
						}
						if mcpErr.Message != tt.errMsg {
							t.Errorf("Expected error message %q, got %q", tt.errMsg, mcpErr.Message)
						}
					} else {
						t.Errorf("Expected a JSON-RPC error of type *types.ErrorResponse, got %T: %v", err, err)
					}
				} else {
					// The transport did NOT return a top-level error, so we do have a response object:
					if resp.Error == nil {
						t.Error("Expected JSON-RPC error in resp.Error, but got nil")
					} else {
						if resp.Error.Code != tt.errCode {
							t.Errorf("Expected error code %d, got %d", tt.errCode, resp.Error.Code)
						}
						if resp.Error.Message != tt.errMsg {
							t.Errorf("Expected error message %q, got %q", tt.errMsg, resp.Error.Message)
						}
					}
				}
				return
			}

			// HAPPY-PATH CASES (tt.wantErr == false)
			// At this point, err must be nil and resp is not nil:
			var result types.CreateMessageResult
			if err := json.Unmarshal(*resp.Result, &result); err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}

			if result.Role != tt.want.Role {
				t.Errorf("Role mismatch: want %s, got %s", tt.want.Role, result.Role)
			}

			// Check content
			wantContent, ok := tt.want.Content.(types.TextContent)
			if !ok {
				t.Fatal("Expected TextContent in want")
			}
			gotContent, ok := result.Content.(types.TextContent)
			if !ok {
				t.Fatal("Expected TextContent in result")
			}
			if gotContent.Text != wantContent.Text {
				t.Errorf("Content text mismatch: want %s, got %s", wantContent.Text, gotContent.Text)
			}

			if result.Model != tt.want.Model {
				t.Errorf("Model mismatch: want %s, got %s", tt.want.Model, result.Model)
			}
			if result.StopReason != tt.want.StopReason {
				t.Errorf("StopReason mismatch: want %s, got %s", tt.want.StopReason, result.StopReason)
			}
		})
	}
}
