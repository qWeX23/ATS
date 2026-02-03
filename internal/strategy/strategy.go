package strategy

import "time"

type Action string

const (
	Hold Action = "HOLD"
	Buy  Action = "BUY"
	Sell Action = "SELL"
)

type MarketSnapshot struct {
	Timestamp   time.Time
	Close       float64
	SMA         float64
	PositionQty int
}

type TradeIntent struct {
	Action Action
	Qty    int
	Reason string
}

type Strategy interface {
	Decide(snapshot MarketSnapshot) TradeIntent
}
