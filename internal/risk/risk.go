package risk

import (
	"fmt"
	"time"

	"ats/internal/strategy"
)

type RiskContext struct {
	Now            time.Time
	Price          float64
	PositionQty    int
	OpenOrderCount int
	LastTradeTime  time.Time
	MaxQty         int
	MaxNotional    float64
	Cooldown       time.Duration
	KillSwitch     bool
	ExtendedHours  bool
	OrderType      string
	TimeInForce    string
}

type ApprovedIntent struct {
	Intent strategy.TradeIntent
	Reason string
}

type Gate struct{}

func (g Gate) Evaluate(intent strategy.TradeIntent, ctx RiskContext) (ApprovedIntent, error) {
	if intent.Action == strategy.Hold {
		return ApprovedIntent{Intent: intent, Reason: "hold"}, nil
	}
	if ctx.KillSwitch {
		return ApprovedIntent{}, fmt.Errorf("kill_switch_enabled")
	}
	if ctx.OpenOrderCount > 0 {
		return ApprovedIntent{}, fmt.Errorf("open_order_exists")
	}
	if ctx.Now.Sub(ctx.LastTradeTime) < ctx.Cooldown {
		return ApprovedIntent{}, fmt.Errorf("cooldown_active")
	}
	if intent.Qty <= 0 {
		return ApprovedIntent{}, fmt.Errorf("invalid_quantity")
	}
	if intent.Action == strategy.Buy && intent.Qty+ctx.PositionQty > ctx.MaxQty {
		return ApprovedIntent{}, fmt.Errorf("max_position_exceeded")
	}
	if intent.Action == strategy.Sell && ctx.PositionQty <= 0 {
		return ApprovedIntent{}, fmt.Errorf("no_position_to_sell")
	}
	if ctx.Price*float64(intent.Qty) > ctx.MaxNotional {
		return ApprovedIntent{}, fmt.Errorf("max_notional_exceeded")
	}
	if ctx.ExtendedHours {
		if ctx.OrderType != "limit" || ctx.TimeInForce != "day" {
			return ApprovedIntent{}, fmt.Errorf("extended_hours_requires_limit_day")
		}
	}

	return ApprovedIntent{Intent: intent, Reason: "approved"}, nil
}
