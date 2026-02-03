package risk

import (
	"testing"
	"time"

	"ats/internal/strategy"
)

func TestGateRejectsCooldown(t *testing.T) {
	gate := Gate{}
	intent := strategy.TradeIntent{Action: strategy.Buy, Qty: 1}
	ctx := RiskContext{
		Now:           time.Now(),
		LastTradeTime: time.Now().Add(-30 * time.Second),
		Cooldown:      time.Minute,
		Price:         100,
		MaxQty:        5,
		MaxNotional:   1000,
	}

	if _, err := gate.Evaluate(intent, ctx); err == nil {
		t.Fatalf("expected cooldown rejection")
	}
}

func TestGateRejectsMaxNotional(t *testing.T) {
	gate := Gate{}
	intent := strategy.TradeIntent{Action: strategy.Buy, Qty: 2}
	ctx := RiskContext{
		Now:         time.Now(),
		Price:       100,
		MaxQty:      5,
		MaxNotional: 150,
	}

	if _, err := gate.Evaluate(intent, ctx); err == nil {
		t.Fatalf("expected max notional rejection")
	}
}

func TestGateApprovesValidBuy(t *testing.T) {
	gate := Gate{}
	intent := strategy.TradeIntent{Action: strategy.Buy, Qty: 1}
	ctx := RiskContext{
		Now:         time.Now(),
		Price:       100,
		MaxQty:      5,
		MaxNotional: 500,
	}

	if _, err := gate.Evaluate(intent, ctx); err != nil {
		t.Fatalf("expected approval, got %v", err)
	}
}

func TestGateRejectsExtendedHoursWithoutLimitDay(t *testing.T) {
	gate := Gate{}
	intent := strategy.TradeIntent{Action: strategy.Buy, Qty: 1}
	ctx := RiskContext{
		Now:           time.Now(),
		Price:         100,
		MaxQty:        5,
		MaxNotional:   500,
		ExtendedHours: true,
		OrderType:     "market",
		TimeInForce:   "day",
	}

	if _, err := gate.Evaluate(intent, ctx); err == nil {
		t.Fatalf("expected extended hours rejection")
	}
}
