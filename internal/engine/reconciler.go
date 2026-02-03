package engine

import (
	"context"
	"errors"
	"log"
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
	orders, err := brokerClient.OpenOrders(ctx)
	if err != nil {
		log.Printf("reconcile open orders failed: %v", err)
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
	}

	position, err := brokerClient.Position(ctx, symbol)
	if err != nil {
		var apiErr *alpaca.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			store.UpdatePosition(state.Position{Qty: 0, AvgEntry: 0})
		} else {
			log.Printf("reconcile position failed: %v", err)
		}
	} else {
		store.UpdatePosition(state.Position{Qty: position.Qty, AvgEntry: position.AvgEntry})
	}

	account, err := brokerClient.Account(ctx)
	if err != nil {
		log.Printf("reconcile account failed: %v", err)
	} else {
		log.Printf("account equity=%.2f buying_power=%.2f", account.Equity, account.BuyingPower)
	}
}
