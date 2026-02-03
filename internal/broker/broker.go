package broker

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpaca-trade-api-go/v3/alpaca"
)

type OrderRequest struct {
	Symbol        string
	Qty           int
	Side          alpaca.Side
	Type          alpaca.OrderType
	TimeInForce   alpaca.TimeInForce
	ClientOrderID string
	ExtendedHours bool
	LimitPrice    *float64
}

type OrderRef struct {
	ID            string
	ClientOrderID string
	Status        string
}

type Position struct {
	Symbol   string
	Qty      int
	AvgEntry float64
}

type Account struct {
	Equity      float64
	BuyingPower float64
}

type Client struct {
	client *alpaca.Client
}

func New(apiKey, apiSecret, baseURL string) *Client {
	opts := alpaca.ClientOpts{
		APIKey:    apiKey,
		APISecret: apiSecret,
		BaseURL:   baseURL,
	}
	return &Client{client: alpaca.NewClient(opts)}
}

func (c *Client) PlaceOrder(ctx context.Context, req OrderRequest) (OrderRef, error) {
	orderReq := alpaca.PlaceOrderRequest{
		Symbol:        req.Symbol,
		Qty:           req.Qty,
		Side:          req.Side,
		Type:          req.Type,
		TimeInForce:   req.TimeInForce,
		ClientOrderID: req.ClientOrderID,
		ExtendedHours: req.ExtendedHours,
	}
	if req.LimitPrice != nil {
		orderReq.LimitPrice = req.LimitPrice
	}

	order, err := c.client.PlaceOrder(orderReq)
	if err != nil {
		return OrderRef{}, err
	}

	return OrderRef{
		ID:            order.ID,
		ClientOrderID: order.ClientOrderID,
		Status:        string(order.Status),
	}, nil
}

func (c *Client) OpenOrders(ctx context.Context) ([]OrderRef, error) {
	req := alpaca.GetOrdersRequest{
		Status: "open",
	}
	orders, err := c.client.GetOrders(req)
	if err != nil {
		return nil, err
	}
	refs := make([]OrderRef, 0, len(orders))
	for _, order := range orders {
		refs = append(refs, OrderRef{
			ID:            order.ID,
			ClientOrderID: order.ClientOrderID,
			Status:        string(order.Status),
		})
	}
	return refs, nil
}

func (c *Client) Position(ctx context.Context, symbol string) (Position, error) {
	pos, err := c.client.GetPosition(symbol)
	if err != nil {
		return Position{}, err
	}
	qty, err := pos.Qty.Int64()
	if err != nil {
		return Position{}, fmt.Errorf("parse qty: %w", err)
	}
	avgEntry, err := pos.AvgEntryPrice.Float64()
	if err != nil {
		return Position{}, fmt.Errorf("parse avg entry: %w", err)
	}

	return Position{
		Symbol:   pos.Symbol,
		Qty:      int(qty),
		AvgEntry: avgEntry,
	}, nil
}

func (c *Client) Account(ctx context.Context) (Account, error) {
	acct, err := c.client.GetAccount()
	if err != nil {
		return Account{}, err
	}
	equity, err := acct.Equity.Float64()
	if err != nil {
		return Account{}, fmt.Errorf("parse equity: %w", err)
	}
	buyingPower, err := acct.BuyingPower.Float64()
	if err != nil {
		return Account{}, fmt.Errorf("parse buying power: %w", err)
	}

	return Account{Equity: equity, BuyingPower: buyingPower}, nil
}

func WaitForContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
