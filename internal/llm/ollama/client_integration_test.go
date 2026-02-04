package ollama_test

import (
	"context"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"ats/internal/llm"
	"ats/internal/llm/ollama"
)

func TestOllamaIntegration_ToolCalling(t *testing.T) {
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		t.Skip("set OLLAMA_MODEL to run integration test")
	}

	baseURL := os.Getenv("OLLAMA_BASE_URL")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	provider := ollama.New(baseURL, model)

	type addArgs struct {
		X int `json:"x" desc:"First number"`
		Y int `json:"y" desc:"Second number"`
	}

	addTool := llm.FuncTool("add", "Add two integers", func(ctx context.Context, args addArgs) (int, error) {
		return args.X + args.Y, nil
	})

	system := "When a function tool is provided for arithmetic, you must call it before answering."

	// First: validate the model actually emits a tool call for this prompt.
	first, err := provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: system,
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "Compute 5+3. Use the add tool with x=5 and y=3. Do not answer until after calling it."},
		},
		Tools: []llm.Tool{addTool},
	})
	if err != nil {
		t.Fatalf("ollama complete failed: %v", err)
	}
	if len(first.ToolCalls) == 0 {
		t.Skipf("model did not request tool calls (content=%q); try another model/prompt", first.Message.Content)
	}

	// Second: validate our tool loop works end-to-end.
	client := llm.New(provider)
	client.RegisterTool(addTool)

	final, err := client.Complete(ctx,
		"Compute 5+3. Return only the number.",
		llm.WithSystemPrompt(system),
		llm.WithMaxIterations(5),
	)
	if err != nil {
		t.Fatalf("client complete failed: %v", err)
	}

	got, ok := firstInt(final.Message.Content)
	if !ok {
		t.Fatalf("expected an integer in response, got %q", final.Message.Content)
	}
	if got != 8 {
		t.Fatalf("expected 8, got %d (raw=%q)", got, final.Message.Content)
	}
}

func firstInt(s string) (int, bool) {
	re := regexp.MustCompile(`-?\d+`)
	m := re.FindString(s)
	if m == "" {
		return 0, false
	}
	v, err := strconv.Atoi(m)
	if err != nil {
		return 0, false
	}
	return v, true
}
