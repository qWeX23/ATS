package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvSetsValues(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, ".env")
	content := "APCA_API_KEY_ID=abc123\nAPCA_API_SECRET_KEY=shh\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	unsetEnv(t, "APCA_API_KEY_ID")
	unsetEnv(t, "APCA_API_SECRET_KEY")

	if err := loadDotEnv(path); err != nil {
		t.Fatalf("loadDotEnv error: %v", err)
	}

	if got := os.Getenv("APCA_API_KEY_ID"); got != "abc123" {
		t.Fatalf("expected key to be set, got %q", got)
	}
	if got := os.Getenv("APCA_API_SECRET_KEY"); got != "shh" {
		t.Fatalf("expected secret to be set, got %q", got)
	}
}

func TestLoadDotEnvDoesNotOverrideExisting(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, ".env")
	content := "APCA_API_KEY_ID=from_file\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	if err := os.Setenv("APCA_API_KEY_ID", "from_env"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	defer unsetEnv(t, "APCA_API_KEY_ID")

	if err := loadDotEnv(path); err != nil {
		t.Fatalf("loadDotEnv error: %v", err)
	}

	if got := os.Getenv("APCA_API_KEY_ID"); got != "from_env" {
		t.Fatalf("expected env to win, got %q", got)
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset env: %v", err)
	}
}
