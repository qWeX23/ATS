package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ats/internal/broker"
	"ats/internal/config"
	"ats/internal/engine"
	"ats/internal/md"
	"ats/internal/risk"
	"ats/internal/state"
	"ats/internal/strategy"
)

func setupLogger() {
	// Check for JSON format (containers/prod) vs pretty text (local dev)
	// Use LOG_FORMAT=json for JSON output, anything else for pretty text
	format := os.Getenv("LOG_FORMAT")

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		// Pretty text format for local development
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			AddSource:   false,
			ReplaceAttr: dropTimeAttr,
		})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// dropTimeAttr removes the time field from text logs for cleaner local output
func dropTimeAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		return slog.Attr{}
	}
	return a
}

func main() {
	setupLogger()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "error", err)
		os.Exit(1)
	}

	runID := generateRunID()
	slog.Info("generated run_id", "run_id", runID)

	slog.Info("initializing decision logger", "path", cfg.DecisionsPath)
	decisions, err := engine.NewDecisionLogger(cfg.DecisionsPath, runID)
	if err != nil {
		slog.Error("decision logger error", "error", err)
		os.Exit(1)
	}
	slog.Info("decision logger initialized", "path", cfg.DecisionsPath, "run_id", runID)
	defer func() {
		if err := decisions.Close(); err != nil {
			slog.Error("failed to close decision logger", "error", err)
		}
	}()

	store := state.NewStore()
	if err := store.Load(cfg.CheckpointPath); err == nil {
		slog.Info("checkpoint loaded", "path", cfg.CheckpointPath)
	} else {
		slog.Info("no checkpoint found, starting fresh", "path", cfg.CheckpointPath)
	}

	slog.Info("initializing broker client", "base_url", cfg.PaperBaseURL)
	brokerClient := broker.New(cfg.APIKey, cfg.APISecret, cfg.PaperBaseURL)

	slog.Info("initializing strategy", "type", "RandomNoise", "max_qty", cfg.MaxQty)
	strategyImpl := strategy.NewRandomNoise(cfg.MaxQty)

	slog.Info("initializing risk gate")
	gate := risk.Gate{}

	slog.Info("creating trading engine")
	engineImpl := engine.New(cfg, strategyImpl, gate, brokerClient, store, decisions)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChan
		slog.Info("shutdown signal received")
		cancel()
	}()

	if cfg.Mode == config.ModePaper {
		slog.Info("starting reconciliation loop", "interval", cfg.ReconcileInterval)
		go engine.ReconcileLoop(ctx, brokerClient, store, cfg.Symbol, cfg.ReconcileInterval)
	}

	slog.Info("bot starting", "mode", cfg.Mode, "symbol", cfg.Symbol, "feed", cfg.Feed, "run_id", runID)
	slog.Info("connecting to market data", "feed", cfg.Feed, "symbol", cfg.Symbol)
	if err := md.StartStream(ctx, cfg.APIKey, cfg.APISecret, cfg.Feed, cfg.Symbol, func(bar md.Bar) {
		engineImpl.OnBar(ctx, bar)
	}); err != nil && err != context.Canceled {
		slog.Info("market data stream stopped", "error", err)
	} else {
		slog.Info("market data stream ended normally")
	}

	slog.Info("saving checkpoint before shutdown")
	if err := store.Save(cfg.CheckpointPath); err != nil {
		slog.Error("failed to save checkpoint", "error", err)
	}

	slog.Info("bot shutdown complete")
}

func generateRunID() string {
	timestamp := time.Now().UTC().Format("20060102T150405")
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return timestamp
	}
	return timestamp + "-" + hex.EncodeToString(randomBytes)
}
