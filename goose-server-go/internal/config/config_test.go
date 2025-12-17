package config

import (
	"os"
	"testing"
)

func TestDefaultDataDir(t *testing.T) {
	dir := DefaultDataDir()
	if dir == "" {
		t.Error("DefaultDataDir should not return empty string")
	}
}

func TestDefaultConfigDir(t *testing.T) {
	dir := DefaultConfigDir()
	if dir == "" {
		t.Error("DefaultConfigDir should not return empty string")
	}
}

func TestLoad(t *testing.T) {
	// Set up test environment
	os.Setenv("GOOSE_SERVER__SECRET_KEY", "test-secret")
	os.Setenv("GOOSE_PORT", "4000")
	defer os.Unsetenv("GOOSE_SERVER__SECRET_KEY")
	defer os.Unsetenv("GOOSE_PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.SecretKey != "test-secret" {
		t.Errorf("SecretKey = %q, want %q", cfg.SecretKey, "test-secret")
	}

	if cfg.Port != 4000 {
		t.Errorf("Port = %d, want %d", cfg.Port, 4000)
	}
}

func TestLoadDefaultPort(t *testing.T) {
	// Clear port env
	os.Unsetenv("GOOSE_PORT")
	os.Setenv("GOOSE_SERVER__SECRET_KEY", "test-secret")
	defer os.Unsetenv("GOOSE_SERVER__SECRET_KEY")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Port != 3000 {
		t.Errorf("Port = %d, want default %d", cfg.Port, 3000)
	}
}

func TestSessionsDBPath(t *testing.T) {
	cfg := &Config{DataDir: "/tmp/goose"}
	path := cfg.SessionsDBPath()
	expected := "/tmp/goose/sessions.db"
	if path != expected {
		t.Errorf("SessionsDBPath = %q, want %q", path, expected)
	}
}

func TestExtensionsDir(t *testing.T) {
	cfg := &Config{DataDir: "/tmp/goose"}
	path := cfg.ExtensionsDir()
	expected := "/tmp/goose/extensions"
	if path != expected {
		t.Errorf("ExtensionsDir = %q, want %q", path, expected)
	}
}

func TestRecipesDir(t *testing.T) {
	cfg := &Config{DataDir: "/tmp/goose"}
	path := cfg.RecipesDir()
	expected := "/tmp/goose/recipes"
	if path != expected {
		t.Errorf("RecipesDir = %q, want %q", path, expected)
	}
}
