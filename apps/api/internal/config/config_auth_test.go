package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadAuthDefaultsInDevelopment(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "postgresql://user:pass@localhost:5432/quotewise")
	t.Setenv("AI_SERVICE_URL", "http://localhost:8000")
	t.Setenv("SESSION_TTL", "")
	t.Setenv("AUTH_RATE_LIMIT_PER_MINUTE", "")
	t.Setenv("RUN_MIGRATIONS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SessionTTL != 168*time.Hour {
		t.Errorf("SessionTTL = %v, want 168h", cfg.SessionTTL)
	}
	if cfg.AuthRateLimitPerMinute != 10 {
		t.Errorf("AuthRateLimitPerMinute = %d, want 10", cfg.AuthRateLimitPerMinute)
	}
	if !cfg.RunMigrations {
		t.Error("RunMigrations = false, want true in development")
	}
}

func TestLoadAuthDefaultsInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgresql://user:pass@db:5432/quotewise")
	t.Setenv("AI_SERVICE_URL", "http://ai:8000")
	t.Setenv("SESSION_TTL", "")
	t.Setenv("AUTH_RATE_LIMIT_PER_MINUTE", "")
	t.Setenv("RUN_MIGRATIONS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RunMigrations {
		t.Error("RunMigrations = true, want false by default in production (opt-in)")
	}
}

func TestLoadAcceptsAuthOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://user:pass@localhost:5432/quotewise")
	t.Setenv("AI_SERVICE_URL", "http://localhost:8000")
	t.Setenv("SESSION_TTL", "24h")
	t.Setenv("AUTH_RATE_LIMIT_PER_MINUTE", "5")
	t.Setenv("RUN_MIGRATIONS", "FALSE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SessionTTL != 24*time.Hour {
		t.Errorf("SessionTTL = %v, want 24h", cfg.SessionTTL)
	}
	if cfg.AuthRateLimitPerMinute != 5 {
		t.Errorf("AuthRateLimitPerMinute = %d, want 5", cfg.AuthRateLimitPerMinute)
	}
	if cfg.RunMigrations {
		t.Error("RunMigrations = true, want false (RUN_MIGRATIONS=FALSE)")
	}
}

func TestLoadRejectsInvalidAuthValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://user:pass@localhost:5432/quotewise")
	t.Setenv("AI_SERVICE_URL", "http://localhost:8000")
	t.Setenv("SESSION_TTL", "forever")
	t.Setenv("AUTH_RATE_LIMIT_PER_MINUTE", "0")
	t.Setenv("RUN_MIGRATIONS", "maybe")

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	for _, want := range []string{"SESSION_TTL", "AUTH_RATE_LIMIT_PER_MINUTE", "RUN_MIGRATIONS"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %s, got: %v", want, err)
		}
	}
}
