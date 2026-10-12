// Package config loads and validates API service configuration from
// the environment. Validation happens at startup so the service fails
// fast with a complete list of problems instead of failing later on
// first use.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the validated runtime configuration for the API service.
type Config struct {
	// Environment is "development", "production", or "test".
	Environment string
	// HTTPPort is the port the HTTP server listens on.
	HTTPPort int
	// LogLevel is one of: debug, info, warn, error.
	LogLevel string
	// LogFormat is "json" (default) or "text" (local debugging).
	LogFormat string
	// DatabaseURL is the PostgreSQL connection string.
	DatabaseURL string
	// AIServiceURL is the base URL of the internal Python AI service.
	AIServiceURL string

	// SessionTTL is how long a login session lasts (sliding expiry;
	// ADR-007).
	SessionTTL time.Duration

	// AuthRateLimitPerMinute bounds auth-endpoint attempts per client
	// IP per minute (in-memory limiter; single instance only).
	AuthRateLimitPerMinute int

	// RunMigrations applies embedded up-only migrations at startup
	// (goose; ADR-016). Default: true in development/test, opt-in in
	// production.
	RunMigrations bool
}

// IsProduction reports whether the service runs in production mode.
func (c Config) IsProduction() bool {
	return c.Environment == "production"
}

// Load reads environment variables and returns a validated Config.
// All problems are reported together so they can be fixed in one pass.
func Load() (Config, error) {
	cfg := Config{
		Environment:  strings.TrimSpace(os.Getenv("APP_ENV")),
		LogLevel:     strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))),
		LogFormat:    strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT"))),
		DatabaseURL:  strings.TrimSpace(os.Getenv("DATABASE_URL")),
		AIServiceURL: strings.TrimSpace(os.Getenv("AI_SERVICE_URL")),
	}

	var problems []string

	if cfg.Environment == "" {
		cfg.Environment = "development"
	} else if !isOneOf(cfg.Environment, "development", "production", "test") {
		problems = append(problems, `APP_ENV must be one of "development", "production", "test"`)
	}

	if rawPort := strings.TrimSpace(os.Getenv("HTTP_PORT")); rawPort == "" {
		cfg.HTTPPort = 8080
	} else {
		port, err := strconv.Atoi(rawPort)
		if err != nil || port < 1 || port > 65535 {
			problems = append(problems, "HTTP_PORT must be an integer between 1 and 65535")
		} else {
			cfg.HTTPPort = port
		}
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	} else if !isOneOf(cfg.LogLevel, "debug", "info", "warn", "error") {
		problems = append(problems, `LOG_LEVEL must be one of "debug", "info", "warn", "error"`)
	}

	if cfg.LogFormat == "" {
		cfg.LogFormat = "json"
	} else if !isOneOf(cfg.LogFormat, "json", "text") {
		problems = append(problems, `LOG_FORMAT must be "json" or "text"`)
	}

	if cfg.DatabaseURL == "" {
		problems = append(problems, "DATABASE_URL is required")
	} else if u, err := url.Parse(cfg.DatabaseURL); err != nil ||
		!isOneOf(u.Scheme, "postgres", "postgresql") || u.Host == "" {
		problems = append(problems, "DATABASE_URL must be a valid postgres:// or postgresql:// URL")
	}

	if cfg.AIServiceURL == "" {
		problems = append(problems, "AI_SERVICE_URL is required")
	} else if u, err := url.Parse(cfg.AIServiceURL); err != nil ||
		!isOneOf(u.Scheme, "http", "https") || u.Host == "" {
		problems = append(problems, "AI_SERVICE_URL must be a valid http:// or https:// URL")
	}

	// --- Authentication (Milestone 2) ---

	if rawTTL := strings.TrimSpace(os.Getenv("SESSION_TTL")); rawTTL == "" {
		cfg.SessionTTL = 168 * time.Hour
	} else {
		ttl, err := time.ParseDuration(rawTTL)
		if err != nil || ttl <= 0 || ttl > 720*time.Hour {
			problems = append(problems, `SESSION_TTL must be a duration like "168h", between 1m and 720h`)
		} else {
			cfg.SessionTTL = ttl
		}
	}

	if rawLimit := strings.TrimSpace(os.Getenv("AUTH_RATE_LIMIT_PER_MINUTE")); rawLimit == "" {
		cfg.AuthRateLimitPerMinute = 10
	} else {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit < 1 || limit > 1000 {
			problems = append(problems, "AUTH_RATE_LIMIT_PER_MINUTE must be an integer between 1 and 1000")
		} else {
			cfg.AuthRateLimitPerMinute = limit
		}
	}

	switch strings.ToLower(strings.TrimSpace(os.Getenv("RUN_MIGRATIONS"))) {
	case "":
		// Safe defaults: apply automatically while developing and
		// testing; in production applying migrations is a deliberate,
		// explicit deployment step.
		cfg.RunMigrations = !cfg.IsProduction()
	case "true":
		cfg.RunMigrations = true
	case "false":
		cfg.RunMigrations = false
	default:
		problems = append(problems, `RUN_MIGRATIONS must be "true" or "false"`)
	}

	if len(problems) > 0 {
		return Config{}, fmt.Errorf("invalid configuration:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return cfg, nil
}

func isOneOf(value string, options ...string) bool {
	for _, option := range options {
		if value == option {
			return true
		}
	}
	return false
}
