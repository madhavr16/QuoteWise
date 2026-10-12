package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// Registers the "pgx" database/sql driver used by the migration
	// runner (the application itself talks to the pool directly).
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/madhavr16/QuoteWise/apps/api/migrations"
)

// Migrate applies all pending up migrations (goose, embedded,
// sequential, up-only by policy). Rollbacks are deliberate, manual
// operations that require explicit approval (ADR-016). The runner uses
// its own database/sql connection so it never competes with the
// application pool.
func (s *Store) Migrate(ctx context.Context) error {
	db, err := sql.Open("pgx", s.databaseURL)
	if err != nil {
		return fmt.Errorf("open migrations connection: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set migration dialect: %w", err)
	}

	migrateCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	// Embedded files live at the FS root (see migrations/embed.go),
	// so the goose directory argument is ".".
	if err := goose.UpContext(migrateCtx, db, "."); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
