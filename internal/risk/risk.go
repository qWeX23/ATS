package risk

import (
	"fmt"
	"log/slog"
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
	notional := ctx.Price * float64(intent.Qty)

	if intent.Action == strategy.Hold {
		return ApprovedIntent{Intent: intent, Reason: "hold"}, nil
	}

	slog.Info("risk evaluation", "intent", intent.Action, "qty", intent.Qty, "position", ctx.PositionQty, "price", ctx.Price, "notional", notional)

	if ctx.KillSwitch {
		slog.Info("risk rejected", "reason", "kill_switch_enabled")
		return ApprovedIntent{}, fmt.Errorf("kill_switch_enabled")
	}
	if ctx.OpenOrderCount > 0 {
		slog.Info("risk rejected", "reason", "open_order_exists", "count", ctx.OpenOrderCount)
		return ApprovedIntent{}, fmt.Errorf("open_order_exists")
	}
	if ctx.Now.Sub(ctx.LastTradeTime) < ctx.Cooldown {
		remaining := ctx.Cooldown - ctx.Now.Sub(ctx.LastTradeTime)
		slog.Info("risk rejected", "reason", "cooldown_active", "remaining", remaining)
		return ApprovedIntent{}, fmt.Errorf("cooldown_active")
	}
	if intent.Qty <= 0 {
		slog.Info("risk rejected", "reason", "invalid_quantity", "qty", intent.Qty)
		return ApprovedIntent{}, fmt.Errorf("invalid_quantity")
	}
	if intent.Action == strategy.Buy && intent.Qty+ctx.PositionQty > ctx.MaxQty {
		slog.Info("risk rejected", "reason", "max_position_exceeded", "new_qty", intent.Qty+ctx.PositionQty, "max", ctx.MaxQty)
		return ApprovedIntent{}, fmt.Errorf("max_position_exceeded")
	}
	if intent.Action == strategy.Sell && ctx.PositionQty <= 0 {
		slog.Info("risk rejected", "reason", "no_position_to_sell")
		return ApprovedIntent{}, fmt.Errorf("no_position_to_sell")
	}
	if notional > ctx.MaxNotional {
		slog.Info("risk rejected", "reason", "max_notional_exceeded", "notional", notional, "max", ctx.MaxNotional)
		return ApprovedIntent{}, fmt.Errorf("max_notional_exceeded")
	}
	if ctx.ExtendedHours {
		if ctx.OrderType != "limit" || ctx.TimeInForce != "day" {
			slog.Info("risk rejected", "reason", "extended_hours_requires_limit_day")
			return ApprovedIntent{}, fmt.Errorf("extended_hours_requires_limit_day")
		}
	}

	slog.Info("risk approved", "intent", intent.Action, "qty", intent.Qty, "reason", intent.Reason)
	return ApprovedIntent{Intent: intent, Reason: "approved"}, nil
}
