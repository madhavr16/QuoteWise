package postgres

import (
	"context"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
)

// CreateOwnerAccount implements the service layer's OwnerAccountCreator
// port: user + organization + owner membership in one transaction
// (ADR-007). The transaction guarantees a half-created account can
// never exist: a failed insert (for example a duplicate email) rolls
// back the organization too.
func (s *Store) CreateOwnerAccount(ctx context.Context, input domain.CreateOwnerAccountInput) (domain.User, domain.Organization, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, domain.Organization{}, err
	}
	// Rollback is a no-op after a successful commit.
	defer tx.Rollback(ctx)

	var organization domain.Organization
	err = tx.QueryRow(ctx, `
		INSERT INTO organizations (name)
		VALUES ($1)
		RETURNING id, name, created_at, updated_at`,
		input.OrganizationName,
	).Scan(&organization.ID, &organization.Name, &organization.CreatedAt, &organization.UpdatedAt)
	if err != nil {
		return domain.User{}, domain.Organization{}, mapQueryError(err)
	}

	var user domain.User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at, updated_at`,
		input.Email, input.PasswordHash,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return domain.User{}, domain.Organization{}, mapQueryError(err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, $3)`,
		organization.ID, user.ID, domain.RoleOwner,
	)
	if err != nil {
		return domain.User{}, domain.Organization{}, mapQueryError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, domain.Organization{}, err
	}
	return user, organization, nil
}
