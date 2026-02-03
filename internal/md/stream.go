package md

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
)

type Bar struct {
	Symbol    string
	Timestamp int64
	Close     float64
}

type BarHandler func(Bar)

func StartStream(ctx context.Context, apiKey, apiSecret, feed, symbol string, handler BarHandler) error {
	opts := marketdata.StreamClientOpts{
		APIKey:    apiKey,
		APISecret: apiSecret,
		Feed:      parseFeed(feed),
	}
	client := marketdata.NewStreamClient(opts)
	client.SubscribeToBars(func(bar marketdata.Bar) {
		handler(Bar{
			Symbol:    bar.Symbol,
			Timestamp: bar.Timestamp.Unix(),
			Close:     bar.Close,
		})
	}, symbol)

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connect market data stream: %w", err)
	}

	<-ctx.Done()
	return ctx.Err()
}

func parseFeed(feed string) marketdata.Feed {
	switch feed {
	case "test":
		return marketdata.Test
	case "iex":
		return marketdata.IEX
	case "sip":
		return marketdata.SIP
	default:
		return marketdata.IEX
	}
}
