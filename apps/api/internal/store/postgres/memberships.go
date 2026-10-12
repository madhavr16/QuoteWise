package postgres

import (
	"context"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
)

// MembershipWithOrganizationByUser returns the user's earliest
// membership with its organization — the single-organization MVP's
// "current organization" (ADR-007); deterministic if multi-membership
// ever lands. Returns domain.ErrNotFound when the user has none.
func (s *Store) MembershipWithOrganizationByUser(ctx context.Context, userID string) (domain.Membership, domain.Organization, error) {
	const query = `
		SELECT m.organization_id, m.user_id, m.role, m.created_at,
		       o.id, o.name, o.created_at, o.updated_at
		FROM organization_members m
		JOIN organizations o ON o.id = m.organization_id
		WHERE m.user_id = $1
		ORDER BY m.created_at, m.organization_id
		LIMIT 1`

	var membership domain.Membership
	var organization domain.Organization
	err := s.pool.QueryRow(ctx, query, userID).Scan(
		&membership.OrganizationID, &membership.UserID, &membership.Role, &membership.CreatedAt,
		&organization.ID, &organization.Name, &organization.CreatedAt, &organization.UpdatedAt,
	)
	if err != nil {
		return domain.Membership{}, domain.Organization{}, mapQueryError(err)
	}
	return membership, organization, nil
}

// MembersByOrganization lists an organization's members (member email
// joined in). Scoped to the given organization only — the caller
// derives it from the session, never from client input (spec §11).
func (s *Store) MembersByOrganization(ctx context.Context, organizationID string) ([]domain.OrganizationMember, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.user_id, u.email, m.role, m.created_at
		FROM organization_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.organization_id = $1
		ORDER BY m.created_at, u.email`,
		organizationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]domain.OrganizationMember, 0)
	for rows.Next() {
		var member domain.OrganizationMember
		if err := rows.Scan(&member.UserID, &member.Email, &member.Role, &member.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}
