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
		{
			name: "Valid request",
			message: types.Message{
				JSONRPC: types.JSONRPCVersion,
				ID:      &types.RequestID{Num: 1},
				Method:  "someMethod",
			},
			wantErr: false,
		},
		{
			name: "Valid notification (no ID)",
			message: types.Message{
				JSONRPC: types.JSONRPCVersion,
				Method:  "notify/something",
			},
			wantErr: false,
		},
		{
			name: "Valid response (result)",
			message: types.Message{
				JSONRPC: types.JSONRPCVersion,
				ID:      &types.RequestID{Num: 2},
				Result:  jsonPtr(`{"ok":true}`),
			},
			wantErr: false,
		},
		{
			name: "Invalid: request with result",
			message: types.Message{
				JSONRPC: types.JSONRPCVersion,
				ID:      &types.RequestID{Num: 3},
				Method:  "badRequest",
				Result:  jsonPtr(`{"some":"thing"}`),
			},
			wantErr: true,
		},
		{
			name: "Invalid: response with both result and error",
			message: types.Message{
				JSONRPC: types.JSONRPCVersion,
				ID:      &types.RequestID{Num: 4},
				Result:  jsonPtr(`{"some":"thing"}`),
				Error:   &types.ErrorResponse{Code: types.InternalError, Message: "oops"},
			},
			wantErr: true,
		},
		{
			name: "Invalid: no method, no result, no error, no ID",
			message: types.Message{
				JSONRPC: types.JSONRPCVersion,
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.message.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() error = %v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func jsonPtr(s string) *json.RawMessage {
	rm := json.RawMessage(s)
	return &rm
}
