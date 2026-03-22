package claude

import (
	"encoding/json"
	"testing"
)

func TestParseResponse_Success(t *testing.T) {
	raw := `{"type":"result","subtype":"success","is_error":false,"result":"ls -la"}`
	var resp Response
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Result != "ls -la" {
		t.Errorf("expected 'ls -la', got '%s'", resp.Result)
	}
	if resp.IsError {
		t.Error("expected is_error to be false")
	}
}

func TestParseResponse_Error(t *testing.T) {
	raw := `{"type":"result","subtype":"error","is_error":true,"result":"something went wrong"}`
	var resp Response
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if !resp.IsError {
		t.Error("expected is_error to be true")
	}
}

func TestParseResponse_Malformed(t *testing.T) {
	raw := `not json at all`
	var resp Response
	if err := json.Unmarshal([]byte(raw), &resp); err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestParseResponse_Empty(t *testing.T) {
	raw := `{"type":"result","subtype":"success","is_error":false,"result":""}`
	var resp Response
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Result != "" {
		t.Errorf("expected empty result, got '%s'", resp.Result)
	}
}
