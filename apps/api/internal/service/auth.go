package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
)

// ErrInvalidCredentials is deliberately generic: it never reveals
// whether the email exists or which part failed (spec §11).
var ErrInvalidCredentials = errors.New("invalid email or password")

// FieldError names one invalid input field.
type FieldError struct {
	Field   string
	Message string
}

func (e FieldError) Error() string { return e.Field + ": " + e.Message }

// ValidationError carries every invalid input field for one request,
// so clients can fix them in a single pass.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, field := range e.Fields {
		parts = append(parts, field.Error())
	}
	return strings.Join(parts, "; ")
}

// AuthContext is the resolved session context (user + their
// organization + role) used for authorization on every request.
type AuthContext struct {
	User         domain.User
	Organization domain.Organization
	Membership   domain.Membership
}

// RegisterInput is validated through domain rules before any store
// call (defense in depth: handlers validate for UX, the use case
// re-validates the invariant).
type RegisterInput struct {
	Email            string
	Password         string
	OrganizationName string
}

// RegisterResult carries the raw session token exactly once — it goes
// into the HttpOnly cookie and is never stored or shown again.
type RegisterResult struct {
	User         domain.User
	Organization domain.Organization
	Token        string
}

// LoginResult is the signed-in user plus the raw session token.
type LoginResult struct {
	User  domain.User
	Token string
}

// AuthService implements the authentication use cases (ADR-007):
// register (creates the organization and owner membership), login,
// session resolution with sliding expiry, and logout.
type AuthService struct {
	ownerAccounts OwnerAccountCreator
	users         UserReader
	memberships   MembershipReader
	sessions      SessionStore
	hasher        PasswordHasher
	tokens        TokenGenerator
	sessionTTL    time.Duration

	dummyOnce sync.Once
	dummyHash string
}

// NewAuthService wires the auth use case. sessionTTL is the sliding
// session lifetime (config-validated).
func NewAuthService(
	ownerAccounts OwnerAccountCreator,
	users UserReader,
	memberships MembershipReader,
	sessions SessionStore,
	hasher PasswordHasher,
	tokens TokenGenerator,
	sessionTTL time.Duration,
) *AuthService {
	return &AuthService{
		ownerAccounts: ownerAccounts,
		users:         users,
		memberships:   memberships,
		sessions:      sessions,
		hasher:        hasher,
		tokens:        tokens,
		sessionTTL:    sessionTTL,
	}
}

// Register creates the user, their organization, the owner membership,
// and a first session.
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (RegisterResult, error) {
	var fields []FieldError
	email, err := domain.NormalizeEmail(input.Email)
	if err != nil {
		fields = append(fields, FieldError{Field: "email", Message: err.Error()})
	}
	if err := domain.ValidatePassword(input.Password); err != nil {
		fields = append(fields, FieldError{Field: "password", Message: err.Error()})
	}
	organizationName := strings.TrimSpace(input.OrganizationName)
	if err := domain.ValidateOrganizationName(organizationName); err != nil {
		fields = append(fields, FieldError{Field: "organization_name", Message: err.Error()})
	}
	if len(fields) > 0 {
		return RegisterResult{}, &ValidationError{Fields: fields}
	}

	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return RegisterResult{}, err
	}

	user, organization, err := s.ownerAccounts.CreateOwnerAccount(ctx, domain.CreateOwnerAccountInput{
		Email:            email,
		PasswordHash:     passwordHash,
		OrganizationName: organizationName,
	})
	if err != nil {
		return RegisterResult{}, err
	}

	token, _, err := s.newSession(ctx, user.ID)
	if err != nil {
		return RegisterResult{}, err
	}
	return RegisterResult{User: user, Organization: organization, Token: token}, nil
}

// Login verifies credentials and starts a session. Unknown emails run
// one dummy verification first so response timing does not reveal
// whether the email exists.
func (s *AuthService) Login(ctx context.Context, email, password string) (LoginResult, error) {
	normalized, err := domain.NormalizeEmail(email)
	if err != nil {
		_, _ = s.hasher.Verify(password, s.timingDummyHash())
		return LoginResult{}, ErrInvalidCredentials
	}

	user, err := s.users.UserByEmail(ctx, normalized)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return LoginResult{}, err
		}
		_, _ = s.hasher.Verify(password, s.timingDummyHash())
		return LoginResult{}, ErrInvalidCredentials
	}

	ok, err := s.hasher.Verify(password, user.PasswordHash)
	if err != nil || !ok {
		return LoginResult{}, ErrInvalidCredentials
	}

	token, _, err := s.newSession(ctx, user.ID)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{User: user, Token: token}, nil
}

// ResolveSession maps a raw cookie token to the auth context. Absent
// or expired sessions return domain.ErrNotFound (the caller decides
// whether that means anonymous or unauthorized).
func (s *AuthService) ResolveSession(ctx context.Context, token string) (AuthContext, error) {
	session, err := s.sessions.SessionByTokenHash(ctx, hashSessionToken(token))
	if err != nil {
		return AuthContext{}, err
	}

	if session.Expired(time.Now()) {
		// Best-effort cleanup; the answer stays "not found".
		_ = s.sessions.DeleteSession(ctx, session.ID)
		return AuthContext{}, domain.ErrNotFound
	}

	user, err := s.users.UserByID(ctx, session.UserID)
	if err != nil {
		return AuthContext{}, err
	}

	membership, organization, err := s.memberships.MembershipWithOrganizationByUser(ctx, user.ID)
	if err != nil {
		return AuthContext{}, err
	}

	// Sliding expiry: extend once past half the TTL so active users stay
	// signed in without a write on every request.
	if time.Until(session.ExpiresAt) < s.sessionTTL/2 {
		_ = s.sessions.TouchSession(ctx, session.ID, time.Now().Add(s.sessionTTL))
	}

	return AuthContext{User: user, Organization: organization, Membership: membership}, nil
}

// Logout revokes the session for the given raw token; idempotent.
func (s *AuthService) Logout(ctx context.Context, token string) error {
	session, err := s.sessions.SessionByTokenHash(ctx, hashSessionToken(token))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}
	return s.sessions.DeleteSession(ctx, session.ID)
}

// newSession creates a session row for the user and returns the raw
// token (hashed at rest) plus the stored session.
func (s *AuthService) newSession(ctx context.Context, userID string) (string, domain.Session, error) {
	token, err := s.tokens.Generate()
	if err != nil {
		return "", domain.Session{}, err
	}
	session := domain.Session{
		UserID:    userID,
		TokenHash: hashSessionToken(token),
		ExpiresAt: time.Now().Add(s.sessionTTL),
	}
	if err := s.sessions.CreateSession(ctx, session); err != nil {
		return "", domain.Session{}, err
	}

	// Opportunistic cleanup of expired sessions (best-effort).
	_ = s.sessions.DeleteExpiredSessions(ctx)

	return token, session, nil
}

// timingDummyHash returns a fixed hash to verify against when the user
// does not exist, keeping login timing uniform. Computed once per
// process; failures fall back to a constant so verification still
// runs (and fails) uniformly.
func (s *AuthService) timingDummyHash() string {
	s.dummyOnce.Do(func() {
		hash, err := s.hasher.Hash("quotewise-dummy-password-for-timing")
		if err != nil {
			s.dummyHash = "$argon2id$dummy"
			return
		}
		s.dummyHash = hash
	})
	return s.dummyHash
}

// hashSessionToken derives the stored hash from a raw token: the raw
// token lives only in the cookie, the hash only in the database.
func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
