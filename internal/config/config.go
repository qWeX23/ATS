package config

import (
	"flag"
	"fmt"
	"os"
	"time"
)

type Mode string

const (
	ModeStream Mode = "stream"
	ModePaper  Mode = "paper"
)

type Config struct {
	Mode              Mode
	Symbol            string
	Feed              string
	BarsWindow        int
	SMAWindow         int
	MaxQty            int
	MaxNotional       float64
	Cooldown          time.Duration
	ReconcileInterval time.Duration
	KillSwitch        bool
	ExtendedHours     bool
	OrderType         string
	TimeInForce       string
	DecisionsPath     string
	CheckpointPath    string
	PaperBaseURL      string
	APIKey            string
	APISecret         string
}

func Load() (Config, error) {
	var cfg Config
	var mode string
	var symbol string
	var feed string

	loadDotEnvIfPresent(".env")

	flag.StringVar(&mode, "mode", string(ModeStream), "run mode: stream or paper")
	flag.StringVar(&symbol, "symbol", "", "trading symbol")
	flag.StringVar(&feed, "feed", "", "market data feed: iex or test")
	flag.IntVar(&cfg.BarsWindow, "bars-window", 50, "number of bars in rolling window")
	flag.IntVar(&cfg.SMAWindow, "sma-window", 20, "SMA window length")
	flag.IntVar(&cfg.MaxQty, "max-qty", 1, "max position size")
	flag.Float64Var(&cfg.MaxNotional, "max-notional", 200, "max notional per order")
	flag.DurationVar(&cfg.Cooldown, "cooldown", 120*time.Second, "cooldown between trades")
	flag.DurationVar(&cfg.ReconcileInterval, "reconcile-interval", 10*time.Second, "reconciliation interval")
	flag.BoolVar(&cfg.KillSwitch, "kill-switch", false, "if true, never place orders")
	flag.BoolVar(&cfg.ExtendedHours, "extended-hours", false, "allow extended hours (limit+day only)")
	flag.StringVar(&cfg.OrderType, "order-type", "market", "order type: market or limit")
	flag.StringVar(&cfg.TimeInForce, "time-in-force", "day", "time in force: day")
	flag.StringVar(&cfg.DecisionsPath, "decisions-path", "decisions.ndjson", "path to decisions log")
	flag.StringVar(&cfg.CheckpointPath, "checkpoint-path", "checkpoint.json", "path to checkpoint file")
	flag.StringVar(&cfg.PaperBaseURL, "paper-base-url", "https://paper-api.alpaca.markets", "paper trading base URL")
	flag.Parse()

	cfg.Mode = Mode(mode)
	cfg.Symbol = symbol
	cfg.Feed = feed
	cfg.APIKey = os.Getenv("APCA_API_KEY_ID")
	cfg.APISecret = os.Getenv("APCA_API_SECRET_KEY")

	if cfg.Mode == ModeStream {
		if cfg.Symbol == "" {
			cfg.Symbol = "FAKEPACA"
		}
		if cfg.Feed == "" {
			cfg.Feed = "test"
		}
	}
	if cfg.Mode == ModePaper {
		if cfg.Symbol == "" {
			cfg.Symbol = "AAPL"
		}
		if cfg.Feed == "" {
			cfg.Feed = "iex"
		}
	}

	if err := validate(cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func validate(cfg Config) error {
	if cfg.Mode != ModeStream && cfg.Mode != ModePaper {
		return fmt.Errorf("invalid mode: %s", cfg.Mode)
	}
	if cfg.APIKey == "" || cfg.APISecret == "" {
		if cfg.Mode == ModePaper {
			return fmt.Errorf("APCA_API_KEY_ID and APCA_API_SECRET_KEY are required in paper mode")
		}
	}
	if cfg.SMAWindow <= 1 {
		return fmt.Errorf("sma-window must be > 1")
	}
	if cfg.BarsWindow < cfg.SMAWindow {
		return fmt.Errorf("bars-window must be >= sma-window")
	}
	if cfg.MaxQty <= 0 {
		return fmt.Errorf("max-qty must be > 0")
	}
	if cfg.MaxNotional <= 0 {
		return fmt.Errorf("max-notional must be > 0")
	}
	if cfg.ReconcileInterval <= 0 {
		return fmt.Errorf("reconcile-interval must be > 0")
	}
	if cfg.Cooldown < 0 {
		return fmt.Errorf("cooldown must be >= 0")
	}
	return nil
}
