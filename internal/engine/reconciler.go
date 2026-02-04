package engine

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"ats/internal/broker"
	"ats/internal/state"
	"github.com/alpacahq/alpaca-trade-api-go/v3/alpaca"
)

func ReconcileLoop(ctx context.Context, brokerClient *broker.Client, store *state.Store, symbol string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reconcileOnce(ctx, brokerClient, store, symbol)
		}
	}
}

func reconcileOnce(ctx context.Context, brokerClient *broker.Client, store *state.Store, symbol string) {
	slog.Info("reconciliation started", "symbol", symbol)

	orders, err := brokerClient.OpenOrders(ctx)
	if err != nil {
		slog.Error("reconcile open orders failed", "error", err)
	} else {
		openOrders := make(map[string]state.OpenOrder, len(orders))
		for _, order := range orders {
			openOrders[order.ClientOrderID] = state.OpenOrder{
				ClientOrderID: order.ClientOrderID,
				OrderID:       order.ID,
				Status:        order.Status,
			}
		}
		store.SetOpenOrders(openOrders)
		slog.Info("reconciled open orders", "count", len(openOrders))
	}

	position, err := brokerClient.Position(ctx, symbol)
	if err != nil {
		var apiErr *alpaca.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			slog.Info("reconciled position", "symbol", symbol, "qty", 0, "status", "no_position")
			store.UpdatePosition(state.Position{Qty: 0, AvgEntry: 0})
		} else {
			slog.Error("reconcile position failed", "error", err)
		}
	} else {
		store.UpdatePosition(state.Position{Qty: position.Qty, AvgEntry: position.AvgEntry})
		slog.Info("reconciled position", "symbol", symbol, "qty", position.Qty, "avg_entry", position.AvgEntry)
	}

	account, err := brokerClient.Account(ctx)
	if err != nil {
		slog.Error("reconcile account failed", "error", err)
	} else {
		slog.Info("reconciled account", "equity", account.Equity, "buying_power", account.BuyingPower)
	}
}
