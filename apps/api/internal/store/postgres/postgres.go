// Package postgres is the persistence adapter layer for PostgreSQL
// (the system of record). It implements the service layer's ports
// over pgx — implicitly, via Go structural typing, so this package
// never imports internal/service. The Milestone 2 slice covers goose
// migrations plus users, organizations, memberships, and sessions;
// procurement entities land in Milestone 2 part 2 (see
// docs/implementation-plan.md).
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps the pgx connection pool and the URL it was built from
// (the URL is reused by the migration runner).
type Store struct {
	databaseURL string
	pool        *pgxpool.Pool
}

// New creates the pool without connecting (pgx connects lazily); call
// Ping to verify reachability.
func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Store{databaseURL: databaseURL, pool: pool}, nil
}

// Close releases the pool's connections.
func (s *Store) Close() {
	s.pool.Close()
}

// Ping reports database reachability (satisfies the service layer's
// Pinger port).
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
