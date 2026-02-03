package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
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

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	runID := generateRunID()
	decisions, err := engine.NewDecisionLogger(cfg.DecisionsPath, runID)
	if err != nil {
		log.Fatalf("decision logger error: %v", err)
	}
	defer func() {
		if err := decisions.Close(); err != nil {
			log.Printf("failed to close decision logger: %v", err)
		}
	}()

	store := state.NewStore()
	if err := store.Load(cfg.CheckpointPath); err == nil {
		log.Printf("loaded checkpoint from %s", cfg.CheckpointPath)
	}

	brokerClient := broker.New(cfg.APIKey, cfg.APISecret, cfg.PaperBaseURL)
	strategyImpl := strategy.NewRandomNoise(cfg.MaxQty)
	gate := risk.Gate{}
	engineImpl := engine.New(cfg, strategyImpl, gate, brokerClient, store, decisions)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChan
		log.Printf("shutdown signal received")
		cancel()
	}()

	if cfg.Mode == config.ModePaper {
		go engine.ReconcileLoop(ctx, brokerClient, store, cfg.Symbol, cfg.ReconcileInterval)
	}

	log.Printf("starting bot mode=%s symbol=%s feed=%s", cfg.Mode, cfg.Symbol, cfg.Feed)
	if err := md.StartStream(ctx, cfg.APIKey, cfg.APISecret, cfg.Feed, cfg.Symbol, func(bar md.Bar) {
		engineImpl.OnBar(ctx, bar)
	}); err != nil && err != context.Canceled {
		log.Printf("market data stream stopped: %v", err)
	}

	if err := store.Save(cfg.CheckpointPath); err != nil {
		log.Printf("failed to save checkpoint: %v", err)
	}

	log.Printf("bot shutdown complete")
}

func generateRunID() string {
	timestamp := time.Now().UTC().Format("20060102T150405")
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return timestamp
	}
	return timestamp + "-" + hex.EncodeToString(randomBytes)
}
