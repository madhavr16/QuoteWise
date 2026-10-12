package postgres

import (
	"context"
	"time"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
)

const sessionColumns = "id, user_id, token_hash, created_at, updated_at, expires_at"

// CreateSession stores a new session. Only the token hash is written;
// the raw token never reaches the database (ADR-007). The ID and
// timestamps are filled by PostgreSQL defaults.
func (s *Store) CreateSession(ctx context.Context, session domain.Session) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		session.UserID, session.TokenHash, session.ExpiresAt,
	)
	return err
}

// SessionByTokenHash looks up a session by its token hash (an exact,
// indexed lookup — no timing-sensitive comparison needed because the
// token is high-entropy). Returns domain.ErrNotFound when absent.
func (s *Store) SessionByTokenHash(ctx context.Context, tokenHash string) (domain.Session, error) {
	var session domain.Session
	err := s.pool.QueryRow(ctx, "SELECT "+sessionColumns+" FROM sessions WHERE token_hash = $1", tokenHash).
		Scan(&session.ID, &session.UserID, &session.TokenHash, &session.CreatedAt, &session.UpdatedAt, &session.ExpiresAt)
	if err != nil {
		return domain.Session{}, mapQueryError(err)
	}
	return session, nil
}

// TouchSession extends a session's expiry (sliding sessions).
func (s *Store) TouchSession(ctx context.Context, id string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, "UPDATE sessions SET expires_at = $2, updated_at = now() WHERE id = $1", id, expiresAt)
	return err
}

// DeleteSession revokes one session (logout).
func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM sessions WHERE id = $1", id)
	return err
}

// DeleteExpiredSessions removes expired sessions (opportunistic
// cleanup; invoked by the auth use case).
func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM sessions WHERE expires_at <= now()")
	return err
}
