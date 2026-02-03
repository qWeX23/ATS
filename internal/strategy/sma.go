package strategy

type SMA struct {
	MaxQty int
}

func (s SMA) Decide(snapshot MarketSnapshot) TradeIntent {
	if snapshot.PositionQty == 0 && snapshot.Close > snapshot.SMA {
		return TradeIntent{
			Action: Buy,
			Qty:    min(s.MaxQty, 1),
			Reason: "close_above_sma",
		}
	}
	if snapshot.PositionQty > 0 && snapshot.Close < snapshot.SMA {
		return TradeIntent{
			Action: Sell,
			Qty:    snapshot.PositionQty,
			Reason: "close_below_sma",
		}
	}
	return TradeIntent{Action: Hold, Reason: "no_signal"}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
