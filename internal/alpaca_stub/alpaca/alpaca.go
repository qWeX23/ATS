package alpaca

import "context"

type Side string

type OrderType string

type TimeInForce string

type OrderStatus string

const (
	Buy  Side = "buy"
	Sell Side = "sell"

	Market OrderType = "market"
	Limit  OrderType = "limit"

	Day TimeInForce = "day"
)

type ClientOpts struct {
	APIKey    string
	APISecret string
	BaseURL   string
}

type Client struct{}

func NewClient(opts ClientOpts) *Client {
	_ = opts
	return &Client{}
}

type PlaceOrderRequest struct {
	Symbol        string
	Qty           int
	Side          Side
	Type          OrderType
	TimeInForce   TimeInForce
	ClientOrderID string
	ExtendedHours bool
	LimitPrice    *float64
}

type Order struct {
	ID            string
	ClientOrderID string
	Status        OrderStatus
}

type GetOrdersRequest struct {
	Status string
}

type Decimal struct {
	value float64
}

func NewDecimal(value float64) Decimal {
	return Decimal{value: value}
}

func (d Decimal) Float64() (float64, error) {
	return d.value, nil
}

func (d Decimal) Int64() (int64, error) {
	return int64(d.value), nil
}

type Position struct {
	Symbol        string
	Qty           Decimal
	AvgEntryPrice Decimal
}

type Account struct {
	Equity      Decimal
	BuyingPower Decimal
}

func (c *Client) PlaceOrder(req PlaceOrderRequest) (Order, error) {
	_ = req
	return Order{}, nil
}

func (c *Client) GetOrders(req GetOrdersRequest) ([]Order, error) {
	_ = req
	return []Order{}, nil
}

func (c *Client) GetPosition(symbol string) (Position, error) {
	_ = symbol
	return Position{}, nil
}

func (c *Client) GetAccount() (Account, error) {
	return Account{}, nil
}

func (c *Client) GetAccountContext(ctx context.Context) (Account, error) {
	_ = ctx
	return Account{}, nil
}
