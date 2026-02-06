package strategy

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"ats/internal/llm"
)

type fakeProvider struct {
	responses []*llm.CompletionResponse
	err       error
}

func (f *fakeProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if len(f.responses) == 0 {
		return &llm.CompletionResponse{
			Message: llm.Message{Role: llm.RoleAssistant, Content: "{}"},
		}, nil
	}
	resp := f.responses[0]
	f.responses = f.responses[1:]
	return resp, nil
}

func (f *fakeProvider) SupportsTools() bool {
	return true
}

func TestLLMStrategyDecide_BuyClampsQty(t *testing.T) {
	provider := &fakeProvider{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							ID: "call-1",
							Function: llm.ToolCallFunction{
								Name:      "decide_trade",
								Arguments: json.RawMessage(`{"action":"BUY","qty":5,"reason":"signal"}`),
							},
						},
					},
				},
			},
			{
				Message: llm.Message{Role: llm.RoleAssistant, Content: "done"},
			},
		},
	}
	client := llm.New(provider)
	strategy := NewLLMStrategy(
		client,
		2,
		"",
		"",
		time.Second,
		"Focus on trend",
	)

	intent := strategy.Decide(MarketSnapshot{
		Timestamp:   time.Now(),
		Close:       101.2,
		SMA:         100.1,
		PositionQty: 0,
	})

	if intent.Action != Buy {
		t.Fatalf("expected BUY, got %s", intent.Action)
	}
	if intent.Qty != 2 {
		t.Fatalf("expected qty 2, got %d", intent.Qty)
	}
	if intent.Reason != "signal" {
		t.Fatalf("expected reason signal, got %q", intent.Reason)
	}
}

func TestLLMStrategyDecide_SellClampsToPosition(t *testing.T) {
	provider := &fakeProvider{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							ID: "call-2",
							Function: llm.ToolCallFunction{
								Name:      "decide_trade",
								Arguments: json.RawMessage(`{"action":"sell","qty":10,"reason":"take_profit"}`),
							},
						},
					},
				},
			},
			{
				Message: llm.Message{Role: llm.RoleAssistant, Content: "done"},
			},
		},
	}
	client := llm.New(provider)
	strategy := NewLLMStrategy(
		client,
		5,
		"",
		"",
		0,
		"",
	)

	intent := strategy.Decide(MarketSnapshot{
		Timestamp:   time.Now(),
		Close:       99.2,
		SMA:         100.1,
		PositionQty: 3,
	})

	if intent.Action != Sell {
		t.Fatalf("expected SELL, got %s", intent.Action)
	}
	if intent.Qty != 3 {
		t.Fatalf("expected qty 3, got %d", intent.Qty)
	}
	if intent.Reason != "take_profit" {
		t.Fatalf("expected reason take_profit, got %q", intent.Reason)
	}
}

func TestLLMStrategyDecide_InvalidResponse(t *testing.T) {
	provider := &fakeProvider{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{Role: llm.RoleAssistant, Content: "not-json"},
			},
		},
	}
	client := llm.New(provider)
	strategy := NewLLMStrategy(
		client,
		1,
		"",
		"",
		0,
		"",
	)

	intent := strategy.Decide(MarketSnapshot{
		Timestamp:   time.Now(),
		Close:       100,
		SMA:         100,
		PositionQty: 0,
	})

	if intent.Action != Hold {
		t.Fatalf("expected HOLD, got %s", intent.Action)
	}
	if intent.Reason != "llm_invalid_response" {
		t.Fatalf("expected invalid response reason, got %q", intent.Reason)
	}
}

func TestLLMStrategyDecide_ProviderError(t *testing.T) {
	client := llm.New(&fakeProvider{err: errors.New("boom")})
	strategy := NewLLMStrategy(
		client,
		1,
		"",
		"",
		0,
		"",
	)

	intent := strategy.Decide(MarketSnapshot{
		Timestamp:   time.Now(),
		Close:       100,
		SMA:         100,
		PositionQty: 0,
	})

	if intent.Action != Hold {
		t.Fatalf("expected HOLD, got %s", intent.Action)
	}
	if intent.Reason != "llm_error: boom" {
		t.Fatalf("expected error reason, got %q", intent.Reason)
	}
}

func TestLLMStrategyDecide_PromptTemplateError(t *testing.T) {
	tempFile, err := os.CreateTemp("", "bad-template-*.md")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer func() {
		_ = os.Remove(tempFile.Name())
	}()

	if _, err := tempFile.WriteString("{{"); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("close temp: %v", err)
	}

	provider := &fakeProvider{
		responses: []*llm.CompletionResponse{
			{Message: llm.Message{Role: llm.RoleAssistant, Content: "ignored"}},
		},
	}
	client := llm.New(provider)
	strategy := NewLLMStrategy(
		client,
		1,
		"",
		tempFile.Name(),
		0,
		"",
	)

	intent := strategy.Decide(MarketSnapshot{
		Timestamp:   time.Now(),
		Close:       100,
		SMA:         100,
		PositionQty: 0,
	})

	if intent.Action != Hold {
		t.Fatalf("expected HOLD, got %s", intent.Action)
	}
	if intent.Reason != "llm_prompt_error" {
		t.Fatalf("expected prompt error reason, got %q", intent.Reason)
	}
}
