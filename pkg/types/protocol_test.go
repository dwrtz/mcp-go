package types_test

import (
	"encoding/json"
	"testing"

	"github.com/dwrtz/mcp-go/pkg/types"
)

func TestMessage_Validate(t *testing.T) {
	tests := []struct {
		name    string
		message types.Message
		wantErr bool
	}{
		// Request Messages
		{
			name: "Valid request with numeric ID",
			message: types.Message{
				JSONRPC: types.JSONRPCVersion,
				ID:      &types.ID{Num: 1},
				Method:  "test/method",
				Params:  jsonPtr(`{"key":"value"}`),
			},
			wantErr: false,
		},
		{
			name: "Valid request with string ID",
			message: types.Message{
				JSONRPC: types.JSONRPCVersion,
				ID:      &types.ID{Str: "abc", IsString: true},
				Method:  "test/method",
			},
			wantErr: false,
		},
		{
			name: "Invalid request - missing method",
			message: types.Message{
				JSONRPC: types.JSONRPCVersion,
				ID:      &types.ID{Num: 1},
				Params:  jsonPtr(`{}`),
			},
			wantErr: true,
		},

		// Notification Messages
		{
			name: "Valid notification",
			message: types.Message{
				JSONRPC: types.JSONRPCVersion,
				Method:  "test/notify",
				Params:  jsonPtr(`{"event":"something"}`),
			},
			wantErr: false,
		},

		// Response Messages
		{
			name: "Valid response with result",
			message: types.Message{
				JSONRPC: types.JSONRPCVersion,
				ID:      &types.ID{Num: 1},
				Result:  jsonPtr(`{"status":"success"}`),
			},
			wantErr: false,
		},
		{
			name: "Valid error response",
			message: types.Message{
				JSONRPC: types.JSONRPCVersion,
				ID:      &types.ID{Num: 1},
				Error: &types.ErrorResponse{
					Code:    types.InvalidParams,
					Message: "invalid parameters",
					Data:    map[string]string{"field": "age"},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid response - missing ID",
			message: types.Message{
				JSONRPC: types.JSONRPCVersion,
				Result:  jsonPtr(`{"status":"success"}`),
			},
			wantErr: true,
		},
		{
			name: "Invalid response - both result and error",
			message: types.Message{
				JSONRPC: types.JSONRPCVersion,
				ID:      &types.ID{Num: 1},
				Result:  jsonPtr(`{"status":"success"}`),
				Error: &types.ErrorResponse{
					Code:    types.InvalidParams,
					Message: "invalid parameters",
				},
			},
			wantErr: true,
		},

		// Version Tests
		{
			name: "Invalid jsonrpc version",
			message: types.Message{
				JSONRPC: "1.0",
				Method:  "test/method",
			},
			wantErr: true,
		},
		{
			name: "Missing jsonrpc version",
			message: types.Message{
				Method: "test/method",
			},
			wantErr: true,
		},

		// Edge Cases
		{
			name: "Empty message",
			message: types.Message{
				JSONRPC: types.JSONRPCVersion,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.message.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func jsonPtr(s string) *json.RawMessage {
	rm := json.RawMessage(s)
	return &rm
}
