package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
)

// Store integration tests run against a dedicated test database when
// TEST_DATABASE_URL is set (see README):
//
//	docker compose exec postgres createdb -U quotewise quotewise_test
//	TEST_DATABASE_URL=postgresql://quotewise:quotewise-dev@localhost:5432/quotewise_test \
//	  go test ./internal/store/postgres/
//
// Without the variable every test below is skipped, so plain
// `go test ./...` works offline. Only the test database is ever
// truncated — never a development or production database.

var (
	testStoreOnce sync.Once
	testStore     *Store
	testStoreErr  error
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping store integration tests (see README)")
	}

	testStoreOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		testStore, testStoreErr = New(ctx, databaseURL)
		if testStoreErr == nil {
			// Migrations are applied by the runner under test itself —
			// the same code path the API uses at startup (ADR-016).
			testStoreErr = testStore.Migrate(ctx)
		}
	})
	if testStoreErr != nil {
		t.Fatalf("test store setup failed: %v", testStoreErr)
	}

	// Clean slate per test — the test database only.
	if _, err := testStore.pool.Exec(
		context.Background(),
		"TRUNCATE TABLE sessions, organization_members, users, organizations CASCADE",
	); err != nil {
		t.Fatalf("truncate test database: %v", err)
	}
	return testStore
}

func TestMigrationsAreIdempotent(t *testing.T) {
	store := newTestStore(t)

	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("second Migrate run must be a no-op, got: %v", err)
	}
}

func TestCreateOwnerAccountPersistsUserOrgAndMembership(t *testing.T) {
	store := newTestStore(t)

	user, organization, err := store.CreateOwnerAccount(context.Background(), domain.CreateOwnerAccountInput{
		Email:            "buyer@example.com",
		PasswordHash:     "$argon2id$v=19$m=65536,t=2,p=2$abc$def",
		OrganizationName: "Acme Procurement",
	})
	if err != nil {
		t.Fatalf("CreateOwnerAccount failed: %v", err)
	}

	found, err := store.UserByEmail(context.Background(), "buyer@example.com")
	if err != nil {
		t.Fatalf("UserByEmail failed: %v", err)
	}
	if found.ID != user.ID || found.PasswordHash != user.PasswordHash {
		t.Errorf("stored user = %+v, want %+v", found, user)
	}

	byID, err := store.UserByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("UserByID failed: %v", err)
	}
	if byID.Email != "buyer@example.com" {
		t.Errorf("UserByID email = %q", byID.Email)
	}

	membership, membershipOrg, err := store.MembershipWithOrganizationByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("MembershipWithOrganizationByUser failed: %v", err)
	}
	if membership.Role != domain.RoleOwner {
		t.Errorf("registration role = %q, want owner", membership.Role)
	}
	if membershipOrg.ID != organization.ID {
		t.Errorf("membership org = %q, want %q", membershipOrg.ID, organization.ID)
	}

	members, err := store.MembersByOrganization(context.Background(), organization.ID)
	if err != nil {
		t.Fatalf("MembersByOrganization failed: %v", err)
	}
	if len(members) != 1 || members[0].Email != "buyer@example.com" || members[0].Role != domain.RoleOwner {
		t.Errorf("members = %+v, want one owner member", members)
	}
}

func TestCreateOwnerAccountDuplicateEmailIsTakenAndRollsBack(t *testing.T) {
	store := newTestStore(t)

	first, firstOrg, err := store.CreateOwnerAccount(context.Background(), domain.CreateOwnerAccountInput{
		Email: "buyer@example.com", PasswordHash: "hash-1", OrganizationName: "Acme",
	})
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	_, _, err = store.CreateOwnerAccount(context.Background(), domain.CreateOwnerAccountInput{
		Email: "buyer@example.com", PasswordHash: "hash-2", OrganizationName: "Evil Clone Co",
	})
	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("duplicate email error = %v, want domain.ErrEmailTaken", err)
	}

	// The failed transaction must leave nothing behind.
	var orgCount int
	if err := testStore.pool.QueryRow(context.Background(), "SELECT count(*) FROM organizations").Scan(&orgCount); err != nil {
		t.Fatalf("count organizations: %v", err)
	}
	if orgCount != 1 {
		t.Errorf("organizations count = %d, want 1 (rollback of the failed insert)", orgCount)
	}

	members, err := store.MembersByOrganization(context.Background(), firstOrg.ID)
	if err != nil {
		t.Fatalf("MembersByOrganization failed: %v", err)
	}
	if len(members) != 1 || members[0].UserID != first.ID {
		t.Errorf("members = %+v, want only the original owner", members)
	}
}

func TestMembershipsAreOrganizationScoped(t *testing.T) {
	store := newTestStore(t)

	userA, orgA, err := store.CreateOwnerAccount(context.Background(), domain.CreateOwnerAccountInput{
		Email: "a@example.com", PasswordHash: "hash-a", OrganizationName: "Org A",
	})
	if err != nil {
		t.Fatalf("org A registration failed: %v", err)
	}
	userB, orgB, err := store.CreateOwnerAccount(context.Background(), domain.CreateOwnerAccountInput{
		Email: "b@example.com", PasswordHash: "hash-b", OrganizationName: "Org B",
	})
	if err != nil {
		t.Fatalf("org B registration failed: %v", err)
	}

	// Each organization sees only its own members.
	membersA, err := store.MembersByOrganization(context.Background(), orgA.ID)
	if err != nil {
		t.Fatalf("MembersByOrganization(A) failed: %v", err)
	}
	if len(membersA) != 1 || membersA[0].UserID != userA.ID {
		t.Errorf("org A members = %+v, want only user A", membersA)
	}

	membersB, err := store.MembersByOrganization(context.Background(), orgB.ID)
	if err != nil {
		t.Fatalf("MembersByOrganization(B) failed: %v", err)
	}
	if len(membersB) != 1 || membersB[0].UserID != userB.ID {
		t.Errorf("org B members = %+v, want only user B", membersB)
	}

	// Each user resolves to their own organization.
	_, resolvedB, err := store.MembershipWithOrganizationByUser(context.Background(), userB.ID)
	if err != nil {
		t.Fatalf("MembershipWithOrganizationByUser(B) failed: %v", err)
	}
	if resolvedB.ID != orgB.ID {
		t.Errorf("user B resolved to org %q, want their own %q", resolvedB.ID, orgB.ID)
	}
}

// sameMicrosecond compares timestamps within PostgreSQL's microsecond
// storage precision (Go's time.Time carries nanoseconds, so exact
// equality after a database round-trip is not a meaningful assertion).
func sameMicrosecond(a, b time.Time) bool {
	return a.Sub(b).Abs() <= time.Microsecond
}

func TestSessionLifecycle(t *testing.T) {
	store := newTestStore(t)

	user, _, err := store.CreateOwnerAccount(context.Background(), domain.CreateOwnerAccountInput{
		Email: "buyer@example.com", PasswordHash: "hash", OrganizationName: "Acme",
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	session := domain.Session{
		UserID:    user.ID,
		TokenHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // 64 hex chars
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := store.CreateSession(context.Background(), session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	found, err := store.SessionByTokenHash(context.Background(), session.TokenHash)
	if err != nil {
		t.Fatalf("SessionByTokenHash failed: %v", err)
	}
	if found.UserID != user.ID || !sameMicrosecond(found.ExpiresAt, session.ExpiresAt) {
		t.Errorf("stored session = %+v, want user %q with expiry %v", found, user.ID, session.ExpiresAt)
	}
	if found.ID == "" {
		t.Error("session ID must be filled by the database default")
	}

	newExpiry := time.Now().Add(2 * time.Hour)
	if err := store.TouchSession(context.Background(), found.ID, newExpiry); err != nil {
		t.Fatalf("TouchSession failed: %v", err)
	}
	touched, err := store.SessionByTokenHash(context.Background(), session.TokenHash)
	if err != nil {
		t.Fatalf("SessionByTokenHash after touch failed: %v", err)
	}
	if !sameMicrosecond(touched.ExpiresAt, newExpiry) {
		t.Errorf("touched expiry = %v, want %v", touched.ExpiresAt, newExpiry)
	}

	if err := store.DeleteSession(context.Background(), found.ID); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}
	if _, err := store.SessionByTokenHash(context.Background(), session.TokenHash); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("deleted session lookup = %v, want domain.ErrNotFound", err)
	}
}

func TestDeleteExpiredSessions(t *testing.T) {
	store := newTestStore(t)

	user, _, err := store.CreateOwnerAccount(context.Background(), domain.CreateOwnerAccountInput{
		Email: "buyer@example.com", PasswordHash: "hash", OrganizationName: "Acme",
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	liveHash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	expiredHash := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	for _, seed := range []struct {
		hash   string
		expiry time.Time
	}{
		{liveHash, time.Now().Add(time.Hour)},
		{expiredHash, time.Now().Add(-time.Minute)},
	} {
		if err := store.CreateSession(context.Background(), domain.Session{
			UserID: user.ID, TokenHash: seed.hash, ExpiresAt: seed.expiry,
		}); err != nil {
			t.Fatalf("CreateSession failed: %v", err)
		}
	}

	if err := store.DeleteExpiredSessions(context.Background()); err != nil {
		t.Fatalf("DeleteExpiredSessions failed: %v", err)
	}

	if _, err := store.SessionByTokenHash(context.Background(), expiredHash); !errors.Is(err, domain.ErrNotFound) {
		t.Error("expired session must be deleted")
	}
	if _, err := store.SessionByTokenHash(context.Background(), liveHash); err != nil {
		t.Errorf("live session must survive cleanup: %v", err)
	}
}
