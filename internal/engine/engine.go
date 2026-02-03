package engine

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"ats/internal/broker"
	"ats/internal/config"
	"ats/internal/md"
	"ats/internal/risk"
	"ats/internal/state"
	"ats/internal/strategy"

	"github.com/alpacahq/alpaca-trade-api-go/v3/alpaca"
)

type Engine struct {
	cfg         config.Config
	strategy    strategy.Strategy
	gate        risk.Gate
	broker      *broker.Client
	state       *state.Store
	decisions   *DecisionLogger
	buffer      *md.RingBuffer
	runID       string
	orderSeqNum uint64
}

func New(cfg config.Config, strategy strategy.Strategy, gate risk.Gate, brokerClient *broker.Client, stateStore *state.Store, decisions *DecisionLogger) *Engine {
	return &Engine{
		cfg:       cfg,
		strategy:  strategy,
		gate:      gate,
		broker:    brokerClient,
		state:     stateStore,
		decisions: decisions,
		buffer:    md.NewRingBuffer(cfg.BarsWindow),
		runID:     decisions.RunID(),
	}
}

func (e *Engine) OnBar(ctx context.Context, bar md.Bar) {
	barTime := time.Unix(bar.Timestamp, 0).UTC()
	log.Printf("DEBUG: OnBar called symbol=%s close=%.2f time=%s", bar.Symbol, bar.Close, barTime.Format(time.RFC3339))
	e.buffer.Add(bar.Close)
	e.state.SetLastBarTime(barTime)

	sma, err := e.buffer.SMA(e.cfg.SMAWindow)
	if err != nil {
		// Not enough data for SMA yet, use close price as fallback for strategies that don't need it
		sma = bar.Close
		log.Printf("bar=%s close=%.2f sma=na (using close as fallback)", barTime.Format(time.RFC3339), bar.Close)
	}

	snapshot := e.state.Snapshot()
	intent := e.strategy.Decide(strategy.MarketSnapshot{
		Timestamp:   barTime,
		Close:       bar.Close,
		SMA:         sma,
		PositionQty: snapshot.Position.Qty,
	})

	riskCtx := risk.RiskContext{
		Now:            time.Now().UTC(),
		Price:          bar.Close,
		PositionQty:    snapshot.Position.Qty,
		OpenOrderCount: len(snapshot.OpenOrders),
		LastTradeTime:  snapshot.LastTradeTime,
		MaxQty:         e.cfg.MaxQty,
		MaxNotional:    e.cfg.MaxNotional,
		Cooldown:       e.cfg.Cooldown,
		KillSwitch:     e.cfg.KillSwitch,
		ExtendedHours:  e.cfg.ExtendedHours,
		OrderType:      e.cfg.OrderType,
		TimeInForce:    e.cfg.TimeInForce,
	}

	approved, err := e.gate.Evaluate(intent, riskCtx)
	decision := Decision{
		RunID:     e.runID,
		Timestamp: time.Now().UTC(),
		BarTime:   barTime,
		Symbol:    bar.Symbol,
		Close:     bar.Close,
		SMA:       sma,
		Intent:    intent.Action,
		IntentQty: intent.Qty,
		Reason:    intent.Reason,
	}

	if err != nil {
		decision.Result = "rejected"
		decision.RejectReason = err.Error()
		e.decisions.Append(decision)
		log.Printf("bar=%s close=%.2f sma=%.2f intent=%s reject=%s", barTime.Format(time.RFC3339), bar.Close, sma, intent.Action, err.Error())
		return
	}

	if intent.Action == strategy.Hold {
		decision.Result = "hold"
		decision.ApprovalReason = approved.Reason
		e.decisions.Append(decision)
		log.Printf("bar=%s close=%.2f sma=%.2f intent=HOLD", barTime.Format(time.RFC3339), bar.Close, sma)
		return
	}

	if e.cfg.Mode == config.ModeStream {
		decision.Result = "dry_run"
		decision.ApprovalReason = approved.Reason
		e.decisions.Append(decision)
		log.Printf("bar=%s close=%.2f sma=%.2f intent=%s dry_run", barTime.Format(time.RFC3339), bar.Close, sma, intent.Action)
		return
	}

	orderReq, err := e.buildOrder(bar.Symbol, bar.Close, approved.Intent)
	if err != nil {
		decision.Result = "order_build_failed"
		decision.RejectReason = err.Error()
		e.decisions.Append(decision)
		log.Printf("bar=%s close=%.2f sma=%.2f intent=%s order_build_failed=%s", barTime.Format(time.RFC3339), bar.Close, sma, intent.Action, err.Error())
		return
	}

	orderRef, err := e.broker.PlaceOrder(ctx, orderReq)
	if err != nil {
		decision.Result = "order_failed"
		decision.RejectReason = err.Error()
		e.decisions.Append(decision)
		log.Printf("bar=%s close=%.2f sma=%.2f intent=%s order_failed=%s", barTime.Format(time.RFC3339), bar.Close, sma, intent.Action, err.Error())
		return
	}

	decision.Result = "order_submitted"
	decision.OrderID = orderRef.ID
	decision.ClientOrderID = orderRef.ClientOrderID
	decision.ApprovalReason = approved.Reason
	e.decisions.Append(decision)
	log.Printf("order_submitted symbol=%s side=%s qty=%d order_id=%s client_order_id=%s", bar.Symbol, intent.Action, intent.Qty, orderRef.ID, orderRef.ClientOrderID)

	e.state.SetLastTradeTime(time.Now().UTC())
	snapshot.OpenOrders[orderRef.ClientOrderID] = state.OpenOrder{
		ClientOrderID: orderRef.ClientOrderID,
		OrderID:       orderRef.ID,
		Status:        orderRef.Status,
	}
	e.state.SetOpenOrders(snapshot.OpenOrders)
}

func (e *Engine) buildOrder(symbol string, price float64, intent strategy.TradeIntent) (broker.OrderRequest, error) {
	orderType, err := parseOrderType(e.cfg.OrderType)
	if err != nil {
		return broker.OrderRequest{}, err
	}
	tif, err := parseTimeInForce(e.cfg.TimeInForce)
	if err != nil {
		return broker.OrderRequest{}, err
	}
	side := alpaca.Buy
	if intent.Action == strategy.Sell {
		side = alpaca.Sell
	}

	req := broker.OrderRequest{
		Symbol:        symbol,
		Qty:           intent.Qty,
		Side:          side,
		Type:          orderType,
		TimeInForce:   tif,
		ClientOrderID: e.nextClientOrderID(),
		ExtendedHours: e.cfg.ExtendedHours,
	}

	if orderType == alpaca.Limit {
		req.LimitPrice = &price
	}

	return req, nil
}

func (e *Engine) nextClientOrderID() string {
	seq := atomic.AddUint64(&e.orderSeqNum, 1)
	return fmt.Sprintf("%s-%d", e.runID, seq)
}

func parseOrderType(value string) (alpaca.OrderType, error) {
	switch value {
	case "market":
		return alpaca.Market, nil
	case "limit":
		return alpaca.Limit, nil
	default:
		return "", fmt.Errorf("unsupported order type: %s", value)
	}
}

func parseTimeInForce(value string) (alpaca.TimeInForce, error) {
	switch value {
	case "day":
		return alpaca.Day, nil
	default:
		return "", fmt.Errorf("unsupported time in force: %s", value)
	}
}
