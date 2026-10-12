package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
)

// mapQueryError normalizes driver errors into domain sentinels — the
// shared kernel every layer may import (ADR-015):
//   - no rows            → domain.ErrNotFound
//   - duplicate email    → domain.ErrEmailTaken
//
// Everything else passes through unchanged; handlers sanitize what
// clients see, and details stay in server-side logs (spec §11).
func mapQueryError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		if pgErr.ConstraintName == "users_email_key" {
			return domain.ErrEmailTaken
		}
	}
	return err
}
