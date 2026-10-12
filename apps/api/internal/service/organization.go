package service

import (
	"context"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
)

// OrganizationService implements organization-management use cases.
// The MVP scope is read-only: registration creates the organization
// and owner; inviting members lands later (open decision P6).
type OrganizationService struct {
	memberships MembershipReader
}

// NewOrganizationService wires the organization use case.
func NewOrganizationService(memberships MembershipReader) *OrganizationService {
	return &OrganizationService{memberships: memberships}
}

// Members lists an organization's members (any member may view).
func (s *OrganizationService) Members(ctx context.Context, organizationID string) ([]domain.OrganizationMember, error) {
	return s.memberships.MembersByOrganization(ctx, organizationID)
}
