package service

import (
	"context"
	"time"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
)

// Ports: outbound interfaces defined where they are consumed (this
// layer) and implemented implicitly by adapters in internal/store and
// internal/security (ADR-015). Every parameter and result type is a
// primitive or a domain type — the shared kernel — so adapters never
// import this package. Tests fake these ports instead of using real
// drivers or crypto.

// Pinger reports dependency reachability (health/readiness).
type Pinger interface {
	Ping(ctx context.Context) error
}

// PasswordHasher hashes passwords and verifies candidates. Hashes are
// PHC-encoded strings so parameters travel with the hash.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, encodedHash string) (bool, error)
}

// TokenGenerator produces unpredictable opaque session tokens.
type TokenGenerator interface {
	Generate() (string, error)
}

// OwnerAccountCreator atomically creates a user, their organization,
// and the owner membership — one store transaction (ADR-007).
// Returns domain.ErrEmailTaken when the email is already registered.
type OwnerAccountCreator interface {
	CreateOwnerAccount(ctx context.Context, input domain.CreateOwnerAccountInput) (domain.User, domain.Organization, error)
}

// UserReader reads user records.
type UserReader interface {
	UserByEmail(ctx context.Context, email string) (domain.User, error)
	UserByID(ctx context.Context, id string) (domain.User, error)
}

// MembershipReader reads organization membership. Every query is
// scoped by user or organization identifiers resolved server-side —
// never by client-supplied organization IDs (spec §11).
type MembershipReader interface {
	// MembershipWithOrganizationByUser returns the user's earliest
	// membership with its organization (the single-organization MVP's
	// "current organization"; ADR-007).
	MembershipWithOrganizationByUser(ctx context.Context, userID string) (domain.Membership, domain.Organization, error)
	// MembersByOrganization lists an organization's members.
	MembersByOrganization(ctx context.Context, organizationID string) ([]domain.OrganizationMember, error)
}

// SessionStore persists login sessions. Only token hashes are ever
// stored (ADR-007); raw tokens live in cookies alone.
type SessionStore interface {
	CreateSession(ctx context.Context, session domain.Session) error
	SessionByTokenHash(ctx context.Context, tokenHash string) (domain.Session, error)
	TouchSession(ctx context.Context, id string, expiresAt time.Time) error
	DeleteSession(ctx context.Context, id string) error
	DeleteExpiredSessions(ctx context.Context) error
}
