package ollama

import (
	"encoding/json"
	"testing"
)

func TestMessageToolNameJSON(t *testing.T) {
	msg := Message{
		Role:     "tool",
		Content:  "ok",
		ToolName: "add",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}

	if payload["tool_name"] != "add" {
		t.Fatalf("expected tool_name 'add', got %v", payload["tool_name"])
	}
	if _, ok := payload["name"]; ok {
		t.Fatal("did not expect name to be set for tool message")
	}
}
