package prompts

import (
	"os"
	"strings"
	"testing"
)

func TestRenderDecisionPrompt_DefaultTemplate(t *testing.T) {
	out, err := RenderDecisionPrompt(DefaultDecisionPrompt(), DecisionData{
		Context:     "Use cautious sizing.",
		Timestamp:   "2024-01-01T00:00:00Z",
		Close:       101.25,
		SMA:         100.0,
		PositionQty: 2,
		MaxQty:      5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Context:") {
		t.Fatalf("expected context header in prompt")
	}
	if !strings.Contains(out, "timestamp=2024-01-01T00:00:00Z") {
		t.Fatalf("expected timestamp in prompt")
	}
}

func TestLoadTemplate_UsesFileOverride(t *testing.T) {
	tempFile, err := os.CreateTemp("", "prompt-*.md")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer func() {
		_ = os.Remove(tempFile.Name())
	}()

	contents := "custom prompt"
	if _, err := tempFile.WriteString(contents); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("close temp: %v", err)
	}

	out := LoadTemplate(tempFile.Name(), "fallback")
	if out != contents {
		t.Fatalf("expected custom prompt, got %q", out)
	}
}
