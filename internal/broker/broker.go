package broker

import (
	"context"
	"time"

	"github.com/alpacahq/alpaca-trade-api-go/v3/alpaca"
	"github.com/shopspring/decimal"
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
	qty := decimal.NewFromInt(int64(req.Qty))
	orderReq := alpaca.PlaceOrderRequest{
		Symbol:        req.Symbol,
		Qty:           &qty,
		Side:          req.Side,
		Type:          req.Type,
		TimeInForce:   req.TimeInForce,
		ClientOrderID: req.ClientOrderID,
		ExtendedHours: req.ExtendedHours,
	}
	if req.LimitPrice != nil {
		limitPrice := decimal.NewFromFloat(*req.LimitPrice)
		orderReq.LimitPrice = &limitPrice
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
	qty := int(pos.Qty.IntPart())
	avgEntry, _ := pos.AvgEntryPrice.Float64()

	return Position{
		Symbol:   pos.Symbol,
		Qty:      qty,
		AvgEntry: avgEntry,
	}, nil
}

func (c *Client) Account(ctx context.Context) (Account, error) {
	acct, err := c.client.GetAccount()
	if err != nil {
		return Account{}, err
	}
	equity, _ := acct.Equity.Float64()
	buyingPower, _ := acct.BuyingPower.Float64()

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
