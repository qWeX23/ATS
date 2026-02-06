package config

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateConfigRejectsInvalidValues(t *testing.T) {
	cfg := Config{
		Mode:              ModePaper,
		APIKey:            "key",
		APISecret:         "secret",
		BarsWindow:        10,
		SMAWindow:         2,
		MaxQty:            0,
		MaxNotional:       100,
		ReconcileInterval: 10,
		Cooldown:          0,
	}

	if err := validate(cfg); err == nil {
		t.Fatalf("expected validation error for max-qty")
	}
}

func TestValidateConfigAcceptsValidConfig(t *testing.T) {
	cfg := Config{
		Mode:              ModeStream,
		BarsWindow:        50,
		SMAWindow:         20,
		MaxQty:            1,
		MaxNotional:       200,
		ReconcileInterval: 10,
		Cooldown:          0,
	}

	if err := validate(cfg); err != nil {
		t.Fatalf("expected config to be valid, got %v", err)
	}
}

func TestLoadConfigPrecedence(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	configContents := `{
  "mode": "stream",
  "strategy": "sma",
  "maxQty": 5,
  "llmModel": "config-model",
  "apiKey": "config-key"
}`
	if err := os.WriteFile(configPath, []byte(configContents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("LLM_MODEL", "env-model")
	t.Setenv("APCA_API_KEY_ID", "env-key")

	resetFlags := resetFlagSet(t)
	defer resetFlags()

	os.Args = []string{
		"cmd",
		"--config", configPath,
		"--strategy", "llm",
		"--max-qty", "2",
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Strategy != "llm" {
		t.Fatalf("expected strategy from CLI, got %q", cfg.Strategy)
	}
	if cfg.MaxQty != 2 {
		t.Fatalf("expected max qty from CLI, got %d", cfg.MaxQty)
	}
	if cfg.LLMModel != "env-model" {
		t.Fatalf("expected LLM model from env, got %q", cfg.LLMModel)
	}
	if cfg.APIKey != "env-key" {
		t.Fatalf("expected API key from env, got %q", cfg.APIKey)
	}
}

func resetFlagSet(t *testing.T) func() {
	t.Helper()
	originalArgs := os.Args
	originalCommandLine := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	return func() {
		flag.CommandLine = originalCommandLine
		os.Args = originalArgs
	}
}
