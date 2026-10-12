package postgres

import (
	"context"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
)

const userColumns = "id, email, password_hash, created_at, updated_at"

// UserByEmail finds a user by their normalized email (ADR-007 login
// path). Returns domain.ErrNotFound when the email is unknown.
func (s *Store) UserByEmail(ctx context.Context, email string) (domain.User, error) {
	return s.scanUser(
		s.pool.QueryRow(ctx, "SELECT "+userColumns+" FROM users WHERE email = $1", email),
	)
}

// UserByID finds a user by ID. Returns domain.ErrNotFound when absent.
func (s *Store) UserByID(ctx context.Context, id string) (domain.User, error) {
	return s.scanUser(
		s.pool.QueryRow(ctx, "SELECT "+userColumns+" FROM users WHERE id = $1", id),
	)
}

func (s *Store) scanUser(row pgxRow) (domain.User, error) {
	var user domain.User
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return domain.User{}, mapQueryError(err)
	}
	return user, nil
}

// pgxRow is satisfied by both pgx.Row and pgx.Tx Row, keeping scan
// helpers testable and tx-friendly.
type pgxRow interface {
	Scan(dest ...any) error
}
