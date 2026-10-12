package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
)

// --- Fakes for the auth ports (fast; no crypto, no I/O) ---

type fakeHasher struct {
	hashErr  error
	verifies int
	verifyOK map[string]bool // password → result; default true
}

func (f *fakeHasher) Hash(password string) (string, error) {
	if f.hashErr != nil {
		return "", f.hashErr
	}
	return "fakehash:" + password, nil
}

func (f *fakeHasher) Verify(password, encodedHash string) (bool, error) {
	f.verifies++
	if ok, found := f.verifyOK[password]; found {
		return ok, nil
	}
	return strings.TrimPrefix(encodedHash, "fakehash:") == password, nil
}

type fakeTokens struct{ counter int }

func (f *fakeTokens) Generate() (string, error) {
	f.counter++
	return "token-" + itoa(f.counter), nil
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

type fakeSessions struct {
	created []domain.Session
	byHash  map[string]domain.Session
	touched []struct {
		id  string
		exp time.Time
	}
	deleted        []string
	expiredDeletes int
}

func (f *fakeSessions) CreateSession(_ context.Context, session domain.Session) error {
	if f.byHash == nil {
		f.byHash = map[string]domain.Session{}
	}
	f.created = append(f.created, session)
	f.byHash[session.TokenHash] = session
	return nil
}

func (f *fakeSessions) SessionByTokenHash(_ context.Context, tokenHash string) (domain.Session, error) {
	if session, ok := f.byHash[tokenHash]; ok {
		return session, nil
	}
	return domain.Session{}, domain.ErrNotFound
}

func (f *fakeSessions) TouchSession(_ context.Context, id string, exp time.Time) error {
	f.touched = append(f.touched, struct {
		id  string
		exp time.Time
	}{id, exp})
	return nil
}

func (f *fakeSessions) DeleteSession(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeSessions) DeleteExpiredSessions(context.Context) error {
	f.expiredDeletes++
	return nil
}

type fakeUsers struct {
	byEmail map[string]domain.User
	byID    map[string]domain.User
}

func (f *fakeUsers) UserByEmail(_ context.Context, email string) (domain.User, error) {
	if user, ok := f.byEmail[email]; ok {
		return user, nil
	}
	return domain.User{}, domain.ErrNotFound
}

func (f *fakeUsers) UserByID(_ context.Context, id string) (domain.User, error) {
	if user, ok := f.byID[id]; ok {
		return user, nil
	}
	return domain.User{}, domain.ErrNotFound
}

type fakeMemberships struct {
	withOrg map[string]struct {
		Membership   domain.Membership
		Organization domain.Organization
	}
	byOrg map[string][]domain.OrganizationMember
}

func (f *fakeMemberships) MembershipWithOrganizationByUser(_ context.Context, userID string) (domain.Membership, domain.Organization, error) {
	if entry, ok := f.withOrg[userID]; ok {
		return entry.Membership, entry.Organization, nil
	}
	return domain.Membership{}, domain.Organization{}, domain.ErrNotFound
}

func (f *fakeMemberships) MembersByOrganization(_ context.Context, organizationID string) ([]domain.OrganizationMember, error) {
	return f.byOrg[organizationID], nil
}

type fakeOwnerAccounts struct {
	created  []domain.CreateOwnerAccountInput
	nextUser int
	failWith map[string]error // email → error
}

func (f *fakeOwnerAccounts) CreateOwnerAccount(_ context.Context, input domain.CreateOwnerAccountInput) (domain.User, domain.Organization, error) {
	if err, ok := f.failWith[input.Email]; ok {
		return domain.User{}, domain.Organization{}, err
	}
	f.created = append(f.created, input)
	f.nextUser++
	user := domain.User{ID: "user-" + itoa(f.nextUser), Email: input.Email, PasswordHash: input.PasswordHash}
	org := domain.Organization{ID: "org-" + itoa(f.nextUser), Name: input.OrganizationName}
	return user, org, nil
}

// --- Test harness ---

const testSessionTTL = time.Hour

type authHarness struct {
	auth     *AuthService
	accounts *fakeOwnerAccounts
	users    *fakeUsers
	members  *fakeMemberships
	sessions *fakeSessions
	hasher   *fakeHasher
	tokens   *fakeTokens
}

func newAuthHarness() *authHarness {
	h := &authHarness{
		accounts: &fakeOwnerAccounts{},
		users:    &fakeUsers{byEmail: map[string]domain.User{}, byID: map[string]domain.User{}},
		members: &fakeMemberships{
			withOrg: map[string]struct {
				Membership   domain.Membership
				Organization domain.Organization
			}{},
			byOrg: map[string][]domain.OrganizationMember{},
		},
		sessions: &fakeSessions{byHash: map[string]domain.Session{}},
		hasher:   &fakeHasher{},
		tokens:   &fakeTokens{},
	}
	h.auth = NewAuthService(h.accounts, h.users, h.members, h.sessions, h.hasher, h.tokens, testSessionTTL)
	return h
}

var ctx = context.Background()

// --- Register ---

func TestRegisterCreatesAccountAndSession(t *testing.T) {
	h := newAuthHarness()

	result, err := h.auth.Register(ctx, RegisterInput{
		Email:            "  Buyer@Example.COM ",
		Password:         "correct horse battery",
		OrganizationName: "  Acme Procurement  ",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Account input is normalized before the store call.
	if len(h.accounts.created) != 1 {
		t.Fatalf("created accounts = %d, want 1", len(h.accounts.created))
	}
	input := h.accounts.created[0]
	if input.Email != "buyer@example.com" {
		t.Errorf("stored email = %q, want normalized %q", input.Email, "buyer@example.com")
	}
	if input.OrganizationName != "Acme Procurement" {
		t.Errorf("stored organization name = %q, want trimmed %q", input.OrganizationName, "Acme Procurement")
	}
	if input.PasswordHash != "fakehash:correct horse battery" {
		t.Errorf("stored hash = %q, want the hasher output", input.PasswordHash)
	}

	// Result carries the raw token; the store keeps only its hash.
	if result.Token != "token-1" {
		t.Errorf("result token = %q, want the generated token", result.Token)
	}
	if len(h.sessions.created) != 1 {
		t.Fatalf("created sessions = %d, want 1", len(h.sessions.created))
	}
	stored := h.sessions.created[0]
	if stored.TokenHash == result.Token {
		t.Error("stored session must hold the token hash, not the raw token")
	}
	if stored.TokenHash != hashSessionToken("token-1") {
		t.Errorf("stored hash = %q, want sha256 of the token", stored.TokenHash)
	}
	if stored.UserID != result.User.ID {
		t.Errorf("session user = %q, want %q", stored.UserID, result.User.ID)
	}
}

func TestRegisterReportsAllValidationErrors(t *testing.T) {
	h := newAuthHarness()

	_, err := h.auth.Register(ctx, RegisterInput{
		Email:            "not-an-email",
		Password:         "short",
		OrganizationName: "",
	})
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("Register error = %v, want ValidationError", err)
	}
	if len(validation.Fields) != 3 {
		t.Fatalf("validation fields = %d, want 3", len(validation.Fields))
	}
	for _, field := range validation.Fields {
		switch field.Field {
		case "email", "password", "organization_name":
		default:
			t.Errorf("unexpected field %q in validation error", field.Field)
		}
	}
	if len(h.accounts.created) != 0 {
		t.Error("no account may be created for invalid input")
	}
}

func TestRegisterPropagatesEmailTaken(t *testing.T) {
	h := newAuthHarness()
	h.accounts.failWith = map[string]error{"buyer@example.com": domain.ErrEmailTaken}

	_, err := h.auth.Register(ctx, RegisterInput{
		Email: "buyer@example.com", Password: "correct horse battery", OrganizationName: "Acme",
	})
	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("Register error = %v, want domain.ErrEmailTaken", err)
	}
	if len(h.sessions.created) != 0 {
		t.Error("no session may be created when registration fails")
	}
}

// --- Login ---

func TestLoginSuccess(t *testing.T) {
	h := newAuthHarness()
	h.users.byEmail["buyer@example.com"] = domain.User{
		ID: "user-1", Email: "buyer@example.com", PasswordHash: "fakehash:correct horse battery",
	}

	result, err := h.auth.Login(ctx, "Buyer@Example.COM", "correct horse battery")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if result.User.ID != "user-1" {
		t.Errorf("logged-in user = %q, want user-1", result.User.ID)
	}
	if result.Token != "token-1" {
		t.Errorf("token = %q, want the generated token", result.Token)
	}
	if len(h.sessions.created) != 1 {
		t.Errorf("created sessions = %d, want 1", len(h.sessions.created))
	}
}

func TestLoginUnknownEmailStillBurnsAVerification(t *testing.T) {
	h := newAuthHarness() // no users

	_, err := h.auth.Login(ctx, "ghost@example.com", "whatever password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login error = %v, want ErrInvalidCredentials", err)
	}
	if h.hasher.verifies == 0 {
		t.Error("unknown-email login must still run one password verification (timing uniformity)")
	}
	if len(h.sessions.created) != 0 {
		t.Error("no session may be created for unknown credentials")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	h := newAuthHarness()
	h.users.byEmail["buyer@example.com"] = domain.User{
		ID: "user-1", Email: "buyer@example.com", PasswordHash: "fakehash:correct horse battery",
	}

	_, err := h.auth.Login(ctx, "buyer@example.com", "wrong password here")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login error = %v, want ErrInvalidCredentials", err)
	}
	if len(h.sessions.created) != 0 {
		t.Error("no session may be created for wrong credentials")
	}
}

// --- ResolveSession ---

func seedSession(h *authHarness, token string, expiresAt time.Time) domain.Session {
	session := domain.Session{ID: "session-" + token, UserID: "user-1", TokenHash: hashSessionToken(token), ExpiresAt: expiresAt}
	h.sessions.byHash[session.TokenHash] = session
	h.users.byID["user-1"] = domain.User{ID: "user-1", Email: "buyer@example.com"}
	h.members.withOrg["user-1"] = struct {
		Membership   domain.Membership
		Organization domain.Organization
	}{domain.Membership{OrganizationID: "org-1", UserID: "user-1", Role: domain.RoleOwner}, domain.Organization{ID: "org-1", Name: "Acme"}}
	return session
}

func TestResolveSessionHappyPath(t *testing.T) {
	h := newAuthHarness()
	seedSession(h, "token-9", time.Now().Add(testSessionTTL))

	authCtx, err := h.auth.ResolveSession(ctx, "token-9")
	if err != nil {
		t.Fatalf("ResolveSession failed: %v", err)
	}
	if authCtx.User.ID != "user-1" || authCtx.Organization.ID != "org-1" || authCtx.Membership.Role != domain.RoleOwner {
		t.Errorf("auth context = %+v, want user-1/org-1/owner", authCtx)
	}
	if len(h.sessions.touched) != 0 {
		t.Error("session far from expiry must not be touched")
	}
}

func TestResolveSessionSlidesWhenNearExpiry(t *testing.T) {
	h := newAuthHarness()
	seedSession(h, "token-9", time.Now().Add(testSessionTTL/4)) // under half the TTL

	_, err := h.auth.ResolveSession(ctx, "token-9")
	if err != nil {
		t.Fatalf("ResolveSession failed: %v", err)
	}
	if len(h.sessions.touched) != 1 {
		t.Fatalf("touches = %d, want 1 (sliding expiry)", len(h.sessions.touched))
	}
}

func TestResolveSessionExpiredDeletesAndReportsNotFound(t *testing.T) {
	h := newAuthHarness()
	session := seedSession(h, "token-9", time.Now().Add(-time.Minute))

	_, err := h.auth.ResolveSession(ctx, "token-9")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("ResolveSession error = %v, want domain.ErrNotFound", err)
	}
	if len(h.sessions.deleted) != 1 || h.sessions.deleted[0] != session.ID {
		t.Errorf("deleted = %v, want [%s] (expired session cleanup)", h.sessions.deleted, session.ID)
	}
}

func TestResolveSessionUnknownTokenIsNotFound(t *testing.T) {
	h := newAuthHarness()

	if _, err := h.auth.ResolveSession(ctx, "never-issued"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("ResolveSession error = %v, want domain.ErrNotFound", err)
	}
}

// --- Logout ---

func TestLogoutDeletesSession(t *testing.T) {
	h := newAuthHarness()
	session := seedSession(h, "token-9", time.Now().Add(testSessionTTL))

	if err := h.auth.Logout(ctx, "token-9"); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}
	if len(h.sessions.deleted) != 1 || h.sessions.deleted[0] != session.ID {
		t.Errorf("deleted = %v, want [%s]", h.sessions.deleted, session.ID)
	}
}

func TestLogoutIsIdempotent(t *testing.T) {
	h := newAuthHarness()

	if err := h.auth.Logout(ctx, "never-issued"); err != nil {
		t.Fatalf("Logout of unknown token = %v, want nil", err)
	}
}
