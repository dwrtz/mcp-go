package testutil

import (
	"encoding/json"
	"testing"

	"github.com/dwrtz/mcp-go/pkg/types"
)

// TestLogger is already defined in logger.go

// CreateTestMessage creates a Message with the given fields
func CreateTestMessage(t *testing.T, id *types.ID, method string, params interface{}) *types.Message {
	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      id,
		Method:  method,
	}

	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal params: %v", err)
		}
		raw := json.RawMessage(data)
		msg.Params = &raw
	}

	return msg
}

// MarshalResult marshals the given result into a json.RawMessage
func MarshalResult(v interface{}) (*json.RawMessage, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	raw := json.RawMessage(data)
	return &raw, nil
}

// CreateTestResult creates a Message containing a result
func CreateTestResult(t *testing.T, id types.ID, result interface{}) *types.Message {
	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      &id,
	}

	if result != nil {
		raw, err := MarshalResult(result)
		if err != nil {
			t.Fatalf("Failed to marshal result: %v", err)
		}
		msg.Result = raw
	}

	return msg
}

// CreateTestError creates a Message containing an error
func CreateTestError(t *testing.T, id types.ID, code int, message string) *types.Message {
	return &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      &id,
		Error: &types.ErrorResponse{
			Code:    code,
			Message: message,
		},
	}
}

// AssertMessagesEqual compares two messages for equality
func AssertMessagesEqual(t *testing.T, expected, actual *types.Message) {
	if expected.JSONRPC != actual.JSONRPC {
		t.Errorf("JSONRPC version mismatch: expected %s, got %s", expected.JSONRPC, actual.JSONRPC)
	}

	if !IDsEqual(expected.ID, actual.ID) {
		t.Errorf("ID mismatch: expected %v, got %v", expected.ID, actual.ID)
	}

	if expected.Method != actual.Method {
		t.Errorf("Method mismatch: expected %s, got %s", expected.Method, actual.Method)
	}

	if !JSONEqual(t, expected.Params, actual.Params) {
		t.Errorf("Params mismatch: expected %s, got %s", expected.Params, actual.Params)
	}

	if !JSONEqual(t, expected.Result, actual.Result) {
		t.Errorf("Result mismatch: expected %s, got %s", expected.Result, actual.Result)
	}

	if !ErrorsEqual(expected.Error, actual.Error) {
		t.Errorf("Error mismatch: expected %v, got %v", expected.Error, actual.Error)
	}
}

// Helper functions

func IDsEqual(a, b *types.ID) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	return a.Num == b.Num && a.Str == b.Str && a.IsString == b.IsString
}

func JSONEqual(t *testing.T, a, b *json.RawMessage) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}

	var va, vb interface{}
	if err := json.Unmarshal(*a, &va); err != nil {
		t.Fatalf("Failed to unmarshal first JSON: %v", err)
	}
	if err := json.Unmarshal(*b, &vb); err != nil {
		t.Fatalf("Failed to unmarshal second JSON: %v", err)
	}

	// Deep equality comparison of unmarshaled values
	return JSONDeepEqual(va, vb)
}

func JSONDeepEqual(a, b interface{}) bool {
	// Convert both to JSON and back to normalize
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)

	var va, vb interface{}
	json.Unmarshal(aJSON, &va)
	json.Unmarshal(bJSON, &vb)

	return JSONDeepEqualValue(va, vb)
}

func JSONDeepEqualValue(a, b interface{}) bool {
	switch va := a.(type) {
	case map[string]interface{}:
		vb, ok := b.(map[string]interface{})
		if !ok {
			return false
		}
		if len(va) != len(vb) {
			return false
		}
		for k, v := range va {
			if !JSONDeepEqualValue(v, vb[k]) {
				return false
			}
		}
		return true
	case []interface{}:
		vb, ok := b.([]interface{})
		if !ok {
			return false
		}
		if len(va) != len(vb) {
			return false
		}
		for i := range va {
			if !JSONDeepEqualValue(va[i], vb[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

func ErrorsEqual(a, b *types.ErrorResponse) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	return a.Code == b.Code && a.Message == b.Message
}
