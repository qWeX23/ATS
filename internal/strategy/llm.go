package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"ats/internal/llm"
	"ats/internal/llm/prompts"
)

type LLMStrategy struct {
	client         *llm.Client
	maxQty         int
	systemPrompt   string
	decisionPrompt string
	timeout        time.Duration
	contextPrompt  string
	mu             sync.Mutex
	lastDecision   *llmDecision
}

type llmDecision struct {
	Action string `json:"action"`
	Qty    int    `json:"qty"`
	Reason string `json:"reason"`
}

func NewLLMStrategy(
	client *llm.Client,
	maxQty int,
	systemPromptPath string,
	decisionPromptPath string,
	timeout time.Duration,
	contextPrompt string,
) *LLMStrategy {
	systemPrompt := prompts.LoadTemplate(systemPromptPath, prompts.DefaultSystemPrompt())
	if systemPromptPath != "" {
		systemPrompt = strings.TrimSpace(systemPrompt)
	}
	decisionPrompt := prompts.LoadTemplate(decisionPromptPath, prompts.DefaultDecisionPrompt())
	strategy := &LLMStrategy{
		client:         client,
		maxQty:         maxQty,
		systemPrompt:   systemPrompt,
		decisionPrompt: decisionPrompt,
		timeout:        timeout,
		contextPrompt:  strings.TrimSpace(contextPrompt),
	}
	strategy.registerTool()
	return strategy
}

func (s *LLMStrategy) Decide(snapshot MarketSnapshot) TradeIntent {
	ctx := context.Background()
	if s.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}

	s.resetDecision()
	prompt, err := prompts.RenderDecisionPrompt(s.decisionPrompt, prompts.DecisionData{
		Context:     s.contextPrompt,
		Timestamp:   snapshot.Timestamp.Format(time.RFC3339),
		Close:       snapshot.Close,
		SMA:         snapshot.SMA,
		PositionQty: snapshot.PositionQty,
		MaxQty:      s.maxQty,
	})
	if err != nil {
		return TradeIntent{Action: Hold, Reason: "llm_prompt_error"}
	}
	resp, err := s.client.Complete(ctx, prompt, llm.WithSystemPrompt(s.systemPrompt))
	if err != nil {
		return TradeIntent{Action: Hold, Reason: fmt.Sprintf("llm_error: %s", err.Error())}
	}

	decision, ok := s.decision()
	if !ok {
		decision, ok = parseLLMDecision(resp.Message.Content)
		if !ok {
			return TradeIntent{Action: Hold, Reason: "llm_invalid_response"}
		}
	}

	action := normalizeAction(decision.Action)
	qty := clampQty(decision.Qty, s.maxQty)
	reason := decision.Reason
	if reason == "" {
		reason = "llm_decision"
	}

	switch action {
	case Buy:
		if qty == 0 {
			return TradeIntent{Action: Hold, Reason: "llm_zero_qty"}
		}
		return TradeIntent{Action: Buy, Qty: qty, Reason: reason}
	case Sell:
		qty = min(qty, snapshot.PositionQty)
		if qty == 0 {
			return TradeIntent{Action: Hold, Reason: "llm_zero_qty"}
		}
		return TradeIntent{Action: Sell, Qty: qty, Reason: reason}
	default:
		return TradeIntent{Action: Hold, Reason: reason}
	}
}

func parseLLMDecision(content string) (llmDecision, bool) {
	var decision llmDecision
	if err := json.Unmarshal([]byte(content), &decision); err == nil {
		return decision, true
	}

	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || end <= start {
		return llmDecision{}, false
	}
	if err := json.Unmarshal([]byte(content[start:end+1]), &decision); err != nil {
		return llmDecision{}, false
	}
	return decision, true
}

func (s *LLMStrategy) registerTool() {
	tool := llm.FuncTool("decide_trade", "Submit a trading decision", func(ctx context.Context, args llmDecision) (map[string]any, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		decision := args
		s.lastDecision = &decision
		return map[string]any{"status": "recorded"}, nil
	})
	s.client.RegisterTool(tool)
}

func (s *LLMStrategy) resetDecision() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastDecision = nil
}

func (s *LLMStrategy) decision() (llmDecision, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastDecision == nil {
		return llmDecision{}, false
	}
	return *s.lastDecision, true
}

func normalizeAction(action string) Action {
	switch strings.ToUpper(strings.TrimSpace(action)) {
	case string(Buy):
		return Buy
	case string(Sell):
		return Sell
	default:
		return Hold
	}
}

func clampQty(qty int, maxQty int) int {
	if qty < 0 {
		return 0
	}
	if qty > maxQty {
		return maxQty
	}
	return qty
}
