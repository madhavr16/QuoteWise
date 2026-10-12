package domain

import "time"

// Role is a member's role in an organization (spec §6).
type Role string

const (
	RoleOwner  Role = "owner"
	RoleMember Role = "member"
)

// Membership links a user to an organization with a role. Complex
// role hierarchies are explicitly out of MVP scope (spec §2).
type Membership struct {
	OrganizationID string
	UserID         string
	Role           Role
	CreatedAt      time.Time
}

// ValidRole reports whether a role value is one of the known roles.
func ValidRole(role Role) bool {
	return role == RoleOwner || role == RoleMember
}

// OrganizationMember is a membership row joined with the user's email
// for member listings. Like CreateOwnerAccountInput, it lives in the
// domain (shared kernel) so adapters can implement membership ports
// without importing the service layer (ADR-015).
type OrganizationMember struct {
	UserID   string
	Email    string
	Role     Role
	JoinedAt time.Time
}
