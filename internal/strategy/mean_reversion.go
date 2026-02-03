package strategy

// MeanReversion implements a Bollinger Bands style strategy
// Buys when price dips below lower band (oversold), sells when it hits upper band (overbought)
type MeanReversion struct {
	MaxQty  int
	BandPct float64 // e.g., 0.02 for 2% bands
	MinBars int     // minimum bars needed before trading
}

func NewMeanReversion(maxQty int) MeanReversion {
	return MeanReversion{
		MaxQty:  maxQty,
		BandPct: 0.015, // 1.5% bands for frequent signals
		MinBars: 10,
	}
}

func (m MeanReversion) Decide(snapshot MarketSnapshot) TradeIntent {
	// Wait for enough data
	if snapshot.SMA == 0 {
		return TradeIntent{Action: Hold, Reason: "insufficient_data"}
	}

	lowerBand := snapshot.SMA * (1 - m.BandPct)
	upperBand := snapshot.SMA * (1 + m.BandPct)

	// Buy: price below lower band and no position
	if snapshot.PositionQty == 0 && snapshot.Close < lowerBand {
		return TradeIntent{
			Action: Buy,
			Qty:    m.MaxQty,
			Reason: "price_below_lower_band",
		}
	}

	// Sell: price above upper band and have position
	if snapshot.PositionQty > 0 && snapshot.Close > upperBand {
		return TradeIntent{
			Action: Sell,
			Qty:    snapshot.PositionQty,
			Reason: "price_above_upper_band",
		}
	}

	// Also sell if price drops below SMA (stop loss / mean reversion failed)
	if snapshot.PositionQty > 0 && snapshot.Close < snapshot.SMA*0.995 {
		return TradeIntent{
			Action: Sell,
			Qty:    snapshot.PositionQty,
			Reason: "stop_loss_below_sma",
		}
	}

	return TradeIntent{Action: Hold, Reason: "within_bands"}
}

// RSIMeanReversion combines RSI oversold signals with mean reversion
type RSIMeanReversion struct {
	MaxQty     int
	RSIPeriod  int
	Oversold   float64   // RSI below this = buy
	Overbought float64   // RSI above this = sell
	rsivalues  []float64 // stores gains/losses for RSI calc
}

func NewRSIMeanReversion(maxQty int) *RSIMeanReversion {
	return &RSIMeanReversion{
		MaxQty:     maxQty,
		RSIPeriod:  7,  // short period for quick signals
		Oversold:   35, // less extreme than 30
		Overbought: 65, // less extreme than 70
		rsivalues:  make([]float64, 0, 20),
	}
}

func (r *RSIMeanReversion) Decide(snapshot MarketSnapshot) TradeIntent {
	// Simple price-based proxy for quick testing
	// In real implementation, you'd calculate actual RSI from price history

	// Mean reversion with tighter bands for frequency
	lowerBand := snapshot.SMA * 0.985 // 1.5% below SMA
	upperBand := snapshot.SMA * 1.015 // 1.5% above SMA

	// Buy signal: price near lower band (oversold)
	if snapshot.PositionQty == 0 && snapshot.Close <= lowerBand {
		return TradeIntent{
			Action: Buy,
			Qty:    r.MaxQty,
			Reason: "oversold_band",
		}
	}

	// Sell signal: price near upper band (overbought)
	if snapshot.PositionQty > 0 && snapshot.Close >= upperBand {
		return TradeIntent{
			Action: Sell,
			Qty:    snapshot.PositionQty,
			Reason: "overbought_band",
		}
	}

	// Quick exit: small profit target (0.5%)
	// This ensures quick turnover for testing
	if snapshot.PositionQty > 0 {
		// Assuming avg entry is around SMA when we bought
		profitTarget := snapshot.SMA * 1.005
		if snapshot.Close >= profitTarget {
			return TradeIntent{
				Action: Sell,
				Qty:    snapshot.PositionQty,
				Reason: "profit_target",
			}
		}
	}

	return TradeIntent{Action: Hold, Reason: "no_signal"}
}

// MomentumStrategy buys on breakout, sells on reversal
// Good for trending markets
type MomentumStrategy struct {
	MaxQty       int
	LookbackBars int
	BreakoutPct  float64
	StopLossPct  float64
	highs        []float64
	lows         []float64
}

func NewMomentumStrategy(maxQty int) *MomentumStrategy {
	return &MomentumStrategy{
		MaxQty:       maxQty,
		LookbackBars: 5,     // very short for quick signals
		BreakoutPct:  0.008, // 0.8% breakout
		StopLossPct:  0.015, // 1.5% stop
		highs:        make([]float64, 0, 20),
		lows:         make([]float64, 0, 20),
	}
}

func (m *MomentumStrategy) Decide(snapshot MarketSnapshot) TradeIntent {
	// Track recent highs and lows using SMA as proxy
	// In production, you'd use actual bar high/low from ring buffer

	recentHigh := snapshot.SMA * 1.005

	// Buy breakout: price breaks above recent high
	if snapshot.PositionQty == 0 && snapshot.Close > recentHigh*(1+m.BreakoutPct) {
		return TradeIntent{
			Action: Buy,
			Qty:    m.MaxQty,
			Reason: "breakout_above_high",
		}
	}

	// Sell: stop loss or momentum reversal
	if snapshot.PositionQty > 0 {
		// Simple momentum reversal - price dropping below SMA
		if snapshot.Close < snapshot.SMA*0.998 {
			return TradeIntent{
				Action: Sell,
				Qty:    snapshot.PositionQty,
				Reason: "momentum_reversal",
			}
		}
	}

	return TradeIntent{Action: Hold, Reason: "consolidating"}
}

// ScalpingStrategy aims for very quick small profits
// Enters on small dips, exits on small gains
type ScalpingStrategy struct {
	MaxQty         int
	EntryThreshold float64 // how far below SMA to enter (e.g., 0.005 = 0.5%)
	ProfitTarget   float64 // exit when up this much (e.g., 0.003 = 0.3%)
	MaxHoldBars    int     // force exit after N bars
	barsInPosition int
}

func NewScalpingStrategy(maxQty int) *ScalpingStrategy {
	return &ScalpingStrategy{
		MaxQty:         maxQty,
		EntryThreshold: 0.008, // 0.8% below SMA
		ProfitTarget:   0.005, // 0.5% profit target
		MaxHoldBars:    5,     // max 5 bars in position
		barsInPosition: 0,
	}
}

func (s *ScalpingStrategy) Decide(snapshot MarketSnapshot) TradeIntent {
	entryPrice := snapshot.SMA * (1 - s.EntryThreshold)
	profitTarget := snapshot.SMA * (1 + s.ProfitTarget)

	// Entry: quick dip
	if snapshot.PositionQty == 0 {
		s.barsInPosition = 0
		if snapshot.Close <= entryPrice {
			return TradeIntent{
				Action: Buy,
				Qty:    s.MaxQty,
				Reason: "scalp_entry",
			}
		}
		return TradeIntent{Action: Hold, Reason: "waiting_for_dip"}
	}

	// Exit logic when in position
	s.barsInPosition++

	// Take profit
	if snapshot.Close >= profitTarget {
		s.barsInPosition = 0
		return TradeIntent{
			Action: Sell,
			Qty:    snapshot.PositionQty,
			Reason: "scalp_profit",
		}
	}

	// Time-based exit (don't hold too long)
	if s.barsInPosition >= s.MaxHoldBars {
		s.barsInPosition = 0
		return TradeIntent{
			Action: Sell,
			Qty:    snapshot.PositionQty,
			Reason: "time_exit",
		}
	}

	// Stop loss (small loss acceptable)
	stopPrice := entryPrice * 0.996 // 0.4% stop
	if snapshot.Close <= stopPrice {
		s.barsInPosition = 0
		return TradeIntent{
			Action: Sell,
			Qty:    snapshot.PositionQty,
			Reason: "scalp_stop",
		}
	}

	return TradeIntent{Action: Hold, Reason: "in_position"}
}

// RandomAlternating buys then sells every tick for testing
// Alternates: Buy -> Sell -> Buy -> Sell...
type RandomAlternating struct {
	MaxQty int
}

func NewRandomAlternating(maxQty int) *RandomAlternating {
	return &RandomAlternating{MaxQty: maxQty}
}

func (r *RandomAlternating) Decide(snapshot MarketSnapshot) TradeIntent {
	// Alternate buy/sell based on current position
	if snapshot.PositionQty == 0 {
		return TradeIntent{
			Action: Buy,
			Qty:    r.MaxQty,
			Reason: "test_buy",
		}
	}
	return TradeIntent{
		Action: Sell,
		Qty:    snapshot.PositionQty,
		Reason: "test_sell",
	}
}

// RandomNoise randomly buys, sells, or holds each tick
// Good for stress testing order submission
type RandomNoise struct {
	MaxQty     int
	BuyProb    float64 // probability of buy (0-1)
	SellProb   float64 // probability of sell (0-1)
	HoldProb   float64 // probability of hold (0-1)
	tradeCount int
}

func NewRandomNoise(maxQty int) *RandomNoise {
	return &RandomNoise{
		MaxQty:   maxQty,
		BuyProb:  0.33,
		SellProb: 0.33,
		HoldProb: 0.34,
	}
}

func (r *RandomNoise) Decide(snapshot MarketSnapshot) TradeIntent {
	r.tradeCount++

	// Cycle through buy/sell/hold based on trade count
	mod := r.tradeCount % 3

	switch mod {
	case 0:
		if snapshot.PositionQty == 0 {
			return TradeIntent{
				Action: Buy,
				Qty:    r.MaxQty,
				Reason: "random_buy",
			}
		}
		return TradeIntent{
			Action: Sell,
			Qty:    snapshot.PositionQty,
			Reason: "random_sell",
		}
	case 1:
		return TradeIntent{Action: Hold, Reason: "random_hold"}
	case 2:
		// Flip position
		if snapshot.PositionQty > 0 {
			return TradeIntent{
				Action: Sell,
				Qty:    snapshot.PositionQty,
				Reason: "random_flip_sell",
			}
		}
		return TradeIntent{
			Action: Buy,
			Qty:    r.MaxQty,
			Reason: "random_flip_buy",
		}
	}

	return TradeIntent{Action: Hold, Reason: "random_hold"}
}
