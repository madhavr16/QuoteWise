package config

import (
	"strings"
	"testing"
)

func TestLoadAppliesDefaultsForOptionalValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://user:pass@localhost:5432/quotewise")
	t.Setenv("AI_SERVICE_URL", "http://localhost:8000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Environment != "development" {
		t.Errorf("Environment = %q, want %q", cfg.Environment, "development")
	}
	if cfg.HTTPPort != 8080 {
		t.Errorf("HTTPPort = %d, want 8080", cfg.HTTPPort)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want %q", cfg.LogFormat, "json")
	}
	if cfg.IsProduction() {
		t.Error("IsProduction() = true, want false")
	}
}

func TestLoadReportsAllProblemsTogether(t *testing.T) {
	// Set explicitly to empty so ambient environment cannot leak in.
	t.Setenv("APP_ENV", "staging")
	t.Setenv("HTTP_PORT", "not-a-port")
	t.Setenv("LOG_LEVEL", "verbose")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("AI_SERVICE_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	for _, want := range []string{"APP_ENV", "HTTP_PORT", "LOG_LEVEL", "DATABASE_URL", "AI_SERVICE_URL"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %s, got: %v", want, err)
		}
	}
}

func TestLoadRejectsBadURLSchemes(t *testing.T) {
	t.Setenv("DATABASE_URL", "mysql://localhost:3306/quotewise")
	t.Setenv("AI_SERVICE_URL", "ftp://localhost:8000")

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error should mention DATABASE_URL, got: %v", err)
	}
	if !strings.Contains(err.Error(), "AI_SERVICE_URL") {
		t.Errorf("error should mention AI_SERVICE_URL, got: %v", err)
	}
}

func TestLoadNormalizesAndAcceptsProductionOverrides(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("HTTP_PORT", "9000")
	t.Setenv("LOG_LEVEL", "WARN") // case-insensitive
	t.Setenv("LOG_FORMAT", "text")
	t.Setenv("DATABASE_URL", "postgres://user:pass@db.internal:5432/quotewise")
	t.Setenv("AI_SERVICE_URL", "https://ai.internal:8000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.IsProduction() {
		t.Error("IsProduction() = false, want true")
	}
	if cfg.HTTPPort != 9000 {
		t.Errorf("HTTPPort = %d, want 9000", cfg.HTTPPort)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want %q (normalized to lowercase)", cfg.LogLevel, "warn")
	}
	if cfg.LogFormat != "text" {
		t.Errorf("LogFormat = %q, want %q", cfg.LogFormat, "text")
	}
}
