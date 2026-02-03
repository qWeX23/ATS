package md

import (
	"context"
	"fmt"
	"log"

	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata/stream"
)

type Bar struct {
	Symbol    string
	Timestamp int64
	Close     float64
}

type BarHandler func(Bar)

func StartStream(ctx context.Context, apiKey, apiSecret, feed, symbol string, handler BarHandler) error {
	feedType := parseFeed(feed)
	client := stream.NewStocksClient(
		feedType,
		stream.WithCredentials(apiKey, apiSecret),
	)

	// Note: Connect must be called BEFORE subscribing in this SDK version
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connect market data stream: %w", err)
	}

	log.Printf("DEBUG: connected to stream, subscribing to bars for symbol=%s", symbol)

	if err := client.SubscribeToBars(func(bar stream.Bar) {
		log.Printf("DEBUG: received bar symbol=%s timestamp=%v close=%.2f", bar.Symbol, bar.Timestamp, bar.Close)
		handler(Bar{
			Symbol:    bar.Symbol,
			Timestamp: bar.Timestamp.Unix(),
			Close:     bar.Close,
		})
	}, symbol); err != nil {
		return fmt.Errorf("subscribe to bars: %w", err)
	}

	log.Printf("DEBUG: subscribed to bars for symbol=%s", symbol)

	<-ctx.Done()
	return ctx.Err()
}

func parseFeed(feed string) marketdata.Feed {
	switch feed {
	case "iex":
		return marketdata.IEX
	case "sip":
		return marketdata.SIP
	default:
		return marketdata.IEX
	}
}
