package config

import "testing"

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
