package marketdata

import (
	"context"
	"time"
)

type Feed string

const (
	Test Feed = "test"
	IEX  Feed = "iex"
	SIP  Feed = "sip"
)

type Bar struct {
	Symbol    string
	Timestamp time.Time
	Close     float64
}

type StreamClientOpts struct {
	APIKey    string
	APISecret string
	Feed      Feed
}

type StreamClient struct{}

type BarHandler func(Bar)

func NewStreamClient(opts StreamClientOpts) *StreamClient {
	_ = opts
	return &StreamClient{}
}

func (s *StreamClient) SubscribeToBars(handler BarHandler, symbols ...string) {
	_ = handler
	_ = symbols
}

func (s *StreamClient) Connect(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
