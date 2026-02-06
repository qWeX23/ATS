package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

type Mode string

const (
	ModeStream Mode = "stream"
	ModePaper  Mode = "paper"
)

type Config struct {
	Mode                  Mode
	Symbol                string
	Feed                  string
	Strategy              string
	BarsWindow            int
	SMAWindow             int
	MaxQty                int
	MaxNotional           float64
	Cooldown              time.Duration
	ReconcileInterval     time.Duration
	KillSwitch            bool
	ExtendedHours         bool
	OrderType             string
	TimeInForce           string
	DecisionsPath         string
	CheckpointPath        string
	PaperBaseURL          string
	APIKey                string
	APISecret             string
	LLMBaseURL            string
	LLMModel              string
	LLMSystemPromptPath   string
	LLMDecisionPromptPath string
	LLMContextPrompt      string
	LLMTimeout            time.Duration
}

func Load() (Config, error) {
	cfg := defaultConfig()
	var mode string
	var symbol string
	var feed string
	var strategy string
	configPath := configPathFromArgs(os.Args)

	loadDotEnvIfPresent(".env")

	if err := applyConfigFile(&cfg, configPath); err != nil {
		return cfg, err
	}

	cfg.APIKey = overrideString(cfg.APIKey, os.Getenv("APCA_API_KEY_ID"))
	cfg.APISecret = overrideString(cfg.APISecret, os.Getenv("APCA_API_SECRET_KEY"))
	cfg.LLMBaseURL = overrideString(cfg.LLMBaseURL, os.Getenv("LLM_BASE_URL"))
	cfg.LLMModel = overrideString(cfg.LLMModel, os.Getenv("LLM_MODEL"))
	cfg.LLMSystemPromptPath = overrideString(cfg.LLMSystemPromptPath, os.Getenv("LLM_SYSTEM_PROMPT_PATH"))
	cfg.LLMDecisionPromptPath = overrideString(cfg.LLMDecisionPromptPath, os.Getenv("LLM_DECISION_PROMPT_PATH"))
	cfg.LLMContextPrompt = overrideString(cfg.LLMContextPrompt, os.Getenv("LLM_CONTEXT_PROMPT"))
	cfg.LLMTimeout = durationFromEnv("LLM_TIMEOUT", cfg.LLMTimeout)

	flag.StringVar(&mode, "mode", string(cfg.Mode), "run mode: stream or paper")
	flag.StringVar(&symbol, "symbol", cfg.Symbol, "trading symbol")
	flag.StringVar(&feed, "feed", cfg.Feed, "market data feed: iex or test")
	flag.StringVar(&strategy, "strategy", cfg.Strategy, "strategy: random_noise, mean_reversion, sma, llm")
	flag.StringVar(&configPath, "config", configPath, "path to JSON config file")
	flag.IntVar(&cfg.BarsWindow, "bars-window", cfg.BarsWindow, "number of bars in rolling window")
	flag.IntVar(&cfg.SMAWindow, "sma-window", cfg.SMAWindow, "SMA window length")
	flag.IntVar(&cfg.MaxQty, "max-qty", cfg.MaxQty, "max position size")
	flag.Float64Var(&cfg.MaxNotional, "max-notional", cfg.MaxNotional, "max notional per order")
	flag.DurationVar(&cfg.Cooldown, "cooldown", cfg.Cooldown, "cooldown between trades")
	flag.DurationVar(&cfg.ReconcileInterval, "reconcile-interval", cfg.ReconcileInterval, "reconciliation interval")
	flag.BoolVar(&cfg.KillSwitch, "kill-switch", cfg.KillSwitch, "if true, never place orders")
	flag.BoolVar(&cfg.ExtendedHours, "extended-hours", cfg.ExtendedHours, "allow extended hours (limit+day only)")
	flag.StringVar(&cfg.OrderType, "order-type", cfg.OrderType, "order type: market or limit")
	flag.StringVar(&cfg.TimeInForce, "time-in-force", cfg.TimeInForce, "time in force: day")
	flag.StringVar(&cfg.DecisionsPath, "decisions-path", cfg.DecisionsPath, "path to decisions log")
	flag.StringVar(&cfg.CheckpointPath, "checkpoint-path", cfg.CheckpointPath, "path to checkpoint file")
	flag.StringVar(&cfg.PaperBaseURL, "paper-base-url", cfg.PaperBaseURL, "paper trading base URL")
	flag.Parse()

	cfg.Mode = Mode(mode)
	cfg.Symbol = symbol
	cfg.Feed = feed
	cfg.Strategy = strategy

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
	if cfg.Strategy == "llm" && cfg.LLMModel == "" {
		return fmt.Errorf("llm-model is required when strategy=llm")
	}
	return nil
}

func defaultConfig() Config {
	return Config{
		Mode:              ModeStream,
		Symbol:            "",
		Feed:              "",
		Strategy:          "random_noise",
		BarsWindow:        50,
		SMAWindow:         20,
		MaxQty:            1,
		MaxNotional:       200,
		Cooldown:          120 * time.Second,
		ReconcileInterval: 10 * time.Second,
		KillSwitch:        false,
		ExtendedHours:     false,
		OrderType:         "market",
		TimeInForce:       "day",
		DecisionsPath:     "decisions.ndjson",
		CheckpointPath:    "checkpoint.json",
		PaperBaseURL:      "https://paper-api.alpaca.markets",
		LLMTimeout:        8 * time.Second,
	}
}

func applyConfigFile(cfg *Config, configPath string) error {
	path := configPath
	if path == "" {
		path = "config.json"
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && configPath == "" {
			return nil
		}
		return fmt.Errorf("read config file: %w", err)
	}
	var fileConfig Config
	if err := json.Unmarshal(contents, &fileConfig); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}
	mergeConfig(cfg, fileConfig)
	return nil
}

func mergeConfig(cfg *Config, other Config) {
	cfg.Mode = Mode(overrideString(string(cfg.Mode), string(other.Mode)))
	cfg.Symbol = overrideString(cfg.Symbol, other.Symbol)
	cfg.Feed = overrideString(cfg.Feed, other.Feed)
	cfg.Strategy = overrideString(cfg.Strategy, other.Strategy)
	cfg.BarsWindow = overrideInt(cfg.BarsWindow, other.BarsWindow)
	cfg.SMAWindow = overrideInt(cfg.SMAWindow, other.SMAWindow)
	cfg.MaxQty = overrideInt(cfg.MaxQty, other.MaxQty)
	cfg.MaxNotional = overrideFloat(cfg.MaxNotional, other.MaxNotional)
	cfg.Cooldown = overrideDuration(cfg.Cooldown, other.Cooldown)
	cfg.ReconcileInterval = overrideDuration(cfg.ReconcileInterval, other.ReconcileInterval)
	cfg.KillSwitch = overrideBool(cfg.KillSwitch, other.KillSwitch)
	cfg.ExtendedHours = overrideBool(cfg.ExtendedHours, other.ExtendedHours)
	cfg.OrderType = overrideString(cfg.OrderType, other.OrderType)
	cfg.TimeInForce = overrideString(cfg.TimeInForce, other.TimeInForce)
	cfg.DecisionsPath = overrideString(cfg.DecisionsPath, other.DecisionsPath)
	cfg.CheckpointPath = overrideString(cfg.CheckpointPath, other.CheckpointPath)
	cfg.PaperBaseURL = overrideString(cfg.PaperBaseURL, other.PaperBaseURL)
	cfg.LLMBaseURL = overrideString(cfg.LLMBaseURL, other.LLMBaseURL)
	cfg.LLMModel = overrideString(cfg.LLMModel, other.LLMModel)
	cfg.LLMSystemPromptPath = overrideString(cfg.LLMSystemPromptPath, other.LLMSystemPromptPath)
	cfg.LLMDecisionPromptPath = overrideString(cfg.LLMDecisionPromptPath, other.LLMDecisionPromptPath)
	cfg.LLMContextPrompt = overrideString(cfg.LLMContextPrompt, other.LLMContextPrompt)
	cfg.LLMTimeout = overrideDuration(cfg.LLMTimeout, other.LLMTimeout)
}

func overrideString(current string, candidate string) string {
	if candidate == "" {
		return current
	}
	return candidate
}

func overrideInt(current int, candidate int) int {
	if candidate == 0 {
		return current
	}
	return candidate
}

func overrideFloat(current float64, candidate float64) float64 {
	if candidate == 0 {
		return current
	}
	return candidate
}

func overrideDuration(current time.Duration, candidate time.Duration) time.Duration {
	if candidate == 0 {
		return current
	}
	return candidate
}

func overrideBool(current bool, candidate bool) bool {
	if candidate {
		return true
	}
	return current
}

func durationFromEnv(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func configPathFromArgs(args []string) string {
	for i := 0; i < len(args); i++ {
		value := args[i]
		if value == "--config" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(value, "--config=") {
			return strings.TrimPrefix(value, "--config=")
		}
	}
	return ""
}
