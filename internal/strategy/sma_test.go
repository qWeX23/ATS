package strategy

import "testing"

func TestSMABuySignal(t *testing.T) {
	strat := SMA{MaxQty: 2}
	snapshot := MarketSnapshot{
		Close:       101,
		SMA:         100,
		PositionQty: 0,
	}
	intent := strat.Decide(snapshot)
	if intent.Action != Buy || intent.Qty != 1 {
		t.Fatalf("expected BUY qty=1, got %s qty=%d", intent.Action, intent.Qty)
	}
}

func TestSMASellSignal(t *testing.T) {
	strat := SMA{MaxQty: 2}
	snapshot := MarketSnapshot{
		Close:       99,
		SMA:         100,
		PositionQty: 3,
	}
	intent := strat.Decide(snapshot)
	if intent.Action != Sell || intent.Qty != 3 {
		t.Fatalf("expected SELL qty=3, got %s qty=%d", intent.Action, intent.Qty)
	}
}

func TestSMAHoldSignal(t *testing.T) {
	strat := SMA{MaxQty: 2}
	snapshot := MarketSnapshot{
		Close:       100,
		SMA:         100,
		PositionQty: 1,
	}
	intent := strat.Decide(snapshot)
	if intent.Action != Hold {
		t.Fatalf("expected HOLD, got %s", intent.Action)
	}
}
