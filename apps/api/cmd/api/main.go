// Command api is the composition root of the QuoteWise API service:
// it loads validated configuration, builds the logger, wires the
// layers (adapters → services → handlers → server), and manages the
// process lifecycle with graceful shutdown.
//
// Layering (clean architecture, ADR-015): handler/server (transport) →
// service (use cases) → domain; store/gateway/security (adapters)
// implement the ports the services define. Dependencies point inward
// only. This is the single place that knows about every layer.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/madhavr16/QuoteWise/apps/api/internal/config"
	"github.com/madhavr16/QuoteWise/apps/api/internal/gateway/aiservice"
	"github.com/madhavr16/QuoteWise/apps/api/internal/handler"
	"github.com/madhavr16/QuoteWise/apps/api/internal/gateway/minio"
	"github.com/madhavr16/QuoteWise/apps/api/internal/security"
	"github.com/madhavr16/QuoteWise/apps/api/internal/server"
	"github.com/madhavr16/QuoteWise/apps/api/internal/service"
	"github.com/madhavr16/QuoteWise/apps/api/internal/store/postgres"
	"github.com/madhavr16/QuoteWise/apps/api/internal/version"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api service exited with error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	// Fail fast on invalid configuration with a complete problem list.
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := newLogger(cfg)
	slog.SetDefault(logger)

	// --- Infrastructure adapters (implement service ports) ---

	db, err := postgres.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	ai := aiservice.New(cfg.AIServiceURL)

	// Warm connectivity check only: the service must still start while
	// PostgreSQL is booting; readiness reports the actual state.
	pingCtx, cancelPing := context.WithTimeout(context.Background(), 2*time.Second)
	if err := db.Ping(pingCtx); err != nil {
		logger.Warn("postgres not reachable yet; continuing startup", slog.Any("error", err))
	} else {
		logger.Info("postgres reachable")
	}
	cancelPing()

	// Migrations: embedded goose migrations, up-only (ADR-016). In
	// development they apply at startup; in production they are an
	// explicit, deliberate deployment step (config-validated).
	if cfg.RunMigrations {
		migrateCtx, cancelMigrate := context.WithTimeout(context.Background(), 60*time.Second)
		if err := db.Migrate(migrateCtx); err != nil {
			cancelMigrate()
			return fmt.Errorf("apply migrations: %w", err)
		}
		cancelMigrate()
		logger.Info("migrations applied")
	}

	// --- MinIO object storage gateway (ADR-008) ---
	minIOEndpoint := cfg.MinIOConfig.Endpoint
	minIOUseSSL := cfg.MinIOConfig.UseSSL
	minIOClient, err := minio.NewMinIO(minIOEndpoint, minIOUseSSL, cfg.MinIOConfig.Bucket, logger)
	if err != nil {
		return fmt.Errorf("create minio client: %w", err)
	}

	// --- Application services (use cases) over ports ---

	healthService := service.NewHealthService(map[string]service.Pinger{
		"postgres":   db,
		"ai_service": ai,
	})
	authService := service.NewAuthService(
		db, db, db, db,
		security.NewArgon2Hasher(),
		security.NewRandomTokenGenerator(),
		cfg.SessionTTL,
	)
	orgService := service.NewOrganizationService(db)

	// --- Procurement-request use case ---
	prService := service.NewProcurementRequestService(db, logger)

	// --- Quotation use case ---
	qService := service.NewQuotationService(db, db, minIOClient, logger)

	// --- Transport: thin handlers + server assembly ---

	engine := server.New(server.Deps{
		Config: cfg,
		Logger: logger,
		Auth:   authService,
		Handlers: server.Handlers{
			Health: handler.NewHealthHandler(healthService),
			Auth:   handler.NewAuthHandler(authService, cfg.IsProduction(), cfg.SessionTTL, logger),
			Org:    handler.NewOrgHandler(orgService, logger),
			Requests: handler.NewProcurementRequestHandler(prService, logger),
			Quotations: handler.NewQuotationUploadHandler(qService, logger),
		},
	})

	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.HTTPPort),
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logger.Info("starting api service",
		slog.String("version", version.Version),
		slog.String("environment", cfg.Environment),
		slog.Int("http_port", cfg.HTTPPort),
	)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-stopCh:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	logger.Info("api service stopped")
	return nil
}

func newLogger(cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(cfg.LogLevel)}
	if cfg.LogFormat == "text" {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
