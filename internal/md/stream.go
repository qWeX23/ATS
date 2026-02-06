package md

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata/stream"
)

type Bar struct {
	Symbol    string
	Timestamp int64
	Close     float64
}

type BarHandler func(Bar)

// SDKLogger wraps slog to satisfy the Alpaca SDK's logger interface
type SDKLogger struct{}

func (l SDKLogger) Printf(format string, v ...any) {
	slog.LogAttrs(context.Background(), slog.LevelInfo, fmt.Sprintf(format, v...),
		slog.String("source", "alpaca_sdk"),
	)
}

func (l SDKLogger) Errorf(format string, v ...any) {
	slog.LogAttrs(context.Background(), slog.LevelError, fmt.Sprintf(format, v...),
		slog.String("source", "alpaca_sdk"),
	)
}

func (l SDKLogger) Infof(format string, v ...any) {
	slog.LogAttrs(context.Background(), slog.LevelInfo, fmt.Sprintf(format, v...),
		slog.String("source", "alpaca_sdk"),
	)
}

func (l SDKLogger) Warnf(format string, v ...any) {
	slog.LogAttrs(context.Background(), slog.LevelWarn, fmt.Sprintf(format, v...),
		slog.String("source", "alpaca_sdk"),
	)
}

func StartStream(ctx context.Context, apiKey, apiSecret, feed, symbol string, handler BarHandler) error {
	feedType := parseFeed(feed)
	client := stream.NewStocksClient(
		feedType,
		stream.WithCredentials(apiKey, apiSecret),
		stream.WithLogger(&SDKLogger{}),
	)

	// Note: Connect must be called BEFORE subscribing in this SDK version
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connect market data stream: %w", err)
	}

	slog.Debug("connected to stream, subscribing to bars", "symbol", symbol)

	if err := client.SubscribeToBars(func(bar stream.Bar) {
		slog.Debug("received bar", "symbol", bar.Symbol, "timestamp", bar.Timestamp, "close", bar.Close)
		handler(Bar{
			Symbol:    bar.Symbol,
			Timestamp: bar.Timestamp.Unix(),
			Close:     bar.Close,
		})
	}, symbol); err != nil {
		return fmt.Errorf("subscribe to bars: %w", err)
	}

	slog.Debug("subscribed to bars", "symbol", symbol)

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
