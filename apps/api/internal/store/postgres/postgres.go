// Package postgres is the persistence adapter layer for PostgreSQL
// (the system of record). It implements the service layer's ports
// over pgx — implicitly, via Go structural typing, so this package
// never imports internal/service. Concrete repositories for
// organizations, procurement requests, quotations, etc. land in
// Milestone 2+ (see docs/implementation-plan.md).
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps the pgx connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// New creates the pool without connecting (pgx connects lazily); call
// Ping to verify reachability.
func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
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
