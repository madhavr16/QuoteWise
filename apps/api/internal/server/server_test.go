package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/madhavr16/QuoteWise/apps/api/internal/config"
	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
	"github.com/madhavr16/QuoteWise/apps/api/internal/handler"
	"github.com/madhavr16/QuoteWise/apps/api/internal/service"
	"github.com/madhavr16/QuoteWise/apps/api/internal/version"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

const testSessionTTL = time.Hour

type fakePinger struct {
	err error
}

func (f fakePinger) Ping(context.Context) error { return f.err }

// --- Compact in-memory auth stack (real use cases over fake ports) ---

type testAuthStack struct {
	mu         sync.Mutex
	nextID     int
	users      map[string]domain.User         // email → user
	byID       map[string]domain.User         // id → user
	sessions   map[string]domain.Session      // token hash → session
	orgByUser  map[string]domain.Organization // user id → organization
	orgMembers map[string][]domain.OrganizationMember
}

func newTestAuthStack() *testAuthStack {
	return &testAuthStack{
		users:      map[string]domain.User{},
		byID:       map[string]domain.User{},
		sessions:   map[string]domain.Session{},
		orgByUser:  map[string]domain.Organization{},
		orgMembers: map[string][]domain.OrganizationMember{},
	}
}

func (s *testAuthStack) CreateOwnerAccount(
	_ context.Context, input domain.CreateOwnerAccountInput,
) (domain.User, domain.Organization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.users[input.Email]; exists {
		return domain.User{}, domain.Organization{}, domain.ErrEmailTaken
	}
	s.nextID++
	user := domain.User{
		ID:           fmt.Sprintf("user-%d", s.nextID),
		Email:        input.Email,
		PasswordHash: input.PasswordHash,
	}
	organization := domain.Organization{
		ID:   fmt.Sprintf("org-%d", s.nextID),
		Name: input.OrganizationName,
	}
	s.users[user.Email] = user
	s.byID[user.ID] = user
	s.orgByUser[user.ID] = organization
	s.orgMembers[organization.ID] = append(s.orgMembers[organization.ID], domain.OrganizationMember{
		UserID:   user.ID,
		Email:    user.Email,
		Role:     domain.RoleOwner,
		JoinedAt: time.Now(),
	})
	return user, organization, nil
}

func (s *testAuthStack) UserByEmail(_ context.Context, email string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if user, ok := s.users[email]; ok {
		return user, nil
	}
	return domain.User{}, domain.ErrNotFound
}

func (s *testAuthStack) UserByID(_ context.Context, id string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if user, ok := s.byID[id]; ok {
		return user, nil
	}
	return domain.User{}, domain.ErrNotFound
}

func (s *testAuthStack) MembershipWithOrganizationByUser(
	_ context.Context, userID string,
) (domain.Membership, domain.Organization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	organization, ok := s.orgByUser[userID]
	if !ok {
		return domain.Membership{}, domain.Organization{}, domain.ErrNotFound
	}
	return domain.Membership{
		OrganizationID: organization.ID,
		UserID:         userID,
		Role:           domain.RoleOwner,
	}, organization, nil
}

func (s *testAuthStack) MembersByOrganization(
	_ context.Context, organizationID string,
) ([]domain.OrganizationMember, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.orgMembers[organizationID], nil
}

func (s *testAuthStack) CreateSession(_ context.Context, session domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.TokenHash] = session
	return nil
}

func (s *testAuthStack) SessionByTokenHash(_ context.Context, tokenHash string) (domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if session, ok := s.sessions[tokenHash]; ok {
		return session, nil
	}
	return domain.Session{}, domain.ErrNotFound
}

func (s *testAuthStack) TouchSession(_ context.Context, id string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for hash, session := range s.sessions {
		if session.ID == id {
			session.ExpiresAt = expiresAt
			s.sessions[hash] = session
		}
	}
	return nil
}

func (s *testAuthStack) DeleteSession(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for hash, session := range s.sessions {
		if session.ID == id {
			delete(s.sessions, hash)
		}
	}
	return nil
}

func (s *testAuthStack) DeleteExpiredSessions(_ context.Context) error { return nil }

type fastHasher struct{}

func (fastHasher) Hash(password string) (string, error)    { return "fh:" + password, nil }
func (fastHasher) Verify(password, h string) (bool, error) { return h == "fh:"+password, nil }

type seqTokens struct {
	mu sync.Mutex
	n  int
}

func (t *seqTokens) Generate() (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.n++
	return fmt.Sprintf("tok-%d", t.n), nil
}

// --- Wiring ---

func newTestDeps(checks map[string]service.Pinger, authRateLimit int) Deps {
	stack := newTestAuthStack()
	authService := service.NewAuthService(
		stack, stack, stack, stack,
		fastHasher{}, &seqTokens{}, testSessionTTL,
	)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return Deps{
		Config: config.Config{
			Environment:            "development",
			HTTPPort:               8080,
			LogLevel:               "info",
			LogFormat:              "json",
			DatabaseURL:            "postgresql://user:pass@localhost:5432/quotewise",
			AIServiceURL:           "http://localhost:8000",
			SessionTTL:             testSessionTTL,
			AuthRateLimitPerMinute: authRateLimit,
			RunMigrations:          true,
		},
		Logger: logger,
		Auth:   authService,
		Handlers: Handlers{
			Health: handler.NewHealthHandler(service.NewHealthService(checks)),
			Auth:   handler.NewAuthHandler(authService, false, testSessionTTL, logger),
			Org:    handler.NewOrgHandler(service.NewOrganizationService(stack), logger),
		},
	}
}

func do(engine *gin.Engine, method, path string) (*httptest.ResponseRecorder, string) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	engine.ServeHTTP(rec, req)
	body, _ := io.ReadAll(rec.Body)
	return rec, string(body)
}

// doJSON issues a request with a JSON body and arbitrary headers.
func doJSON(
	engine *gin.Engine, method, path, body string, headers map[string]string,
) (*httptest.ResponseRecorder, string) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	engine.ServeHTTP(rec, req)
	respBody, _ := io.ReadAll(rec.Body)
	return rec, string(respBody)
}

func csrfHeaders() map[string]string {
	return map[string]string{handler.CSRFHeader: handler.CSRFHeaderValue}
}

const (
	registerBody = `{"email":"buyer@example.com","password":"correct horse battery","organization_name":"Acme Procurement"}`
	loginBody    = `{"email":"buyer@example.com","password":"correct horse battery"}`
)

func sessionCookieFrom(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	cookie := rec.Header().Get("Set-Cookie")
	if cookie == "" {
		t.Fatal("response has no Set-Cookie header")
	}
	if !strings.Contains(cookie, "HttpOnly") {
		t.Errorf("session cookie must be HttpOnly: %s", cookie)
	}
	if !strings.Contains(cookie, "SameSite=Lax") {
		t.Errorf("session cookie must be SameSite=Lax: %s", cookie)
	}
	_, value, _ := strings.Cut(strings.SplitN(cookie, ";", 2)[0], "=")
	return value
}

// --- Health (Milestone 1, unchanged behavior) ---

func TestHealthzIsAlwaysOK(t *testing.T) {
	engine := New(newTestDeps(nil, 10))

	rec, body := do(engine, http.MethodGet, "/healthz")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(body, `"status":"ok"`) || !strings.Contains(body, version.Version) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestReadyzPassesWhenAllChecksPass(t *testing.T) {
	engine := New(newTestDeps(map[string]service.Pinger{
		"postgres":   fakePinger{},
		"ai_service": fakePinger{},
	}, 10))

	rec, body := do(engine, http.MethodGet, "/readyz")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(body, `"status":"ready"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestReadyzFailsWhenADependencyFails(t *testing.T) {
	engine := New(newTestDeps(map[string]service.Pinger{
		"postgres":   fakePinger{err: errors.New("connection refused")},
		"ai_service": fakePinger{},
	}, 10))

	rec, body := do(engine, http.MethodGet, "/readyz")
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(body, `"status":"not_ready"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

// --- Auth flow ---

func TestMeWithoutSessionIsUnauthorized(t *testing.T) {
	engine := New(newTestDeps(nil, 10))

	rec, body := do(engine, http.MethodGet, "/api/v1/me")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(body, `"code":"authentication_required"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestRegisterRequiresCSRFForMutations(t *testing.T) {
	engine := New(newTestDeps(nil, 10))

	rec, body := doJSON(engine, http.MethodPost, "/api/v1/auth/register", registerBody, nil)
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d (CSRF header missing)", rec.Code, http.StatusForbidden)
	}
	if !strings.Contains(body, `"code":"csrf_header_required"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestRegisterRejectsInvalidJSON(t *testing.T) {
	engine := New(newTestDeps(nil, 10))

	rec, _ := doJSON(engine, http.MethodPost, "/api/v1/auth/register", "not json", csrfHeaders())
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRegisterReportsValidationErrorsPerField(t *testing.T) {
	engine := New(newTestDeps(nil, 10))

	rec, body := doJSON(engine, http.MethodPost, "/api/v1/auth/register",
		`{"email":"bad","password":"short","organization_name":""}`, csrfHeaders())
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	var envelope handler.ErrorEnvelope
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("body is not the error envelope: %v\nbody: %s", err, body)
	}
	if envelope.Error.Code != "validation_error" {
		t.Errorf("code = %q, want validation_error", envelope.Error.Code)
	}
	for _, field := range []string{"email", "password", "organization_name"} {
		if !strings.Contains(envelope.Error.Message, field) {
			t.Errorf("message should mention %q: %s", field, envelope.Error.Message)
		}
	}
}

func TestRegisterDuplicateEmailIsConflict(t *testing.T) {
	engine := New(newTestDeps(nil, 10))

	rec, _ := doJSON(engine, http.MethodPost, "/api/v1/auth/register", registerBody, csrfHeaders())
	if rec.Code != http.StatusCreated {
		t.Fatalf("first register status = %d, want %d", rec.Code, http.StatusCreated)
	}

	rec, body := doJSON(engine, http.MethodPost, "/api/v1/auth/register", registerBody, csrfHeaders())
	if rec.Code != http.StatusConflict {
		t.Errorf("duplicate register status = %d, want %d", rec.Code, http.StatusConflict)
	}
	if !strings.Contains(body, `"code":"email_taken"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestRegisterLoginLogoutSessionFlow(t *testing.T) {
	engine := New(newTestDeps(nil, 100))

	// Register: 201 + session cookie with security flags; the token
	// never appears in the body.
	rec, body := doJSON(engine, http.MethodPost, "/api/v1/auth/register", registerBody, csrfHeaders())
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if strings.Contains(body, "tok-") {
		t.Error("raw session token must never appear in a response body")
	}
	cookie := sessionCookieFrom(t, rec)
	if cookie == "" {
		t.Fatal("session cookie value is empty")
	}

	// Authenticated request with the cookie.
	authHeaders := map[string]string{"Cookie": handler.SessionCookieName + "=" + cookie}
	rec, body = doJSON(engine, http.MethodGet, "/api/v1/me", "", authHeaders)
	if rec.Code != http.StatusOK {
		t.Fatalf("me status = %d, want %d", rec.Code, http.StatusOK)
	}
	for _, want := range []string{"buyer@example.com", "Acme Procurement", `"role":"owner"`} {
		if !strings.Contains(body, want) {
			t.Errorf("me response should contain %q: %s", want, body)
		}
	}

	// Members listing (organization scoped server-side).
	rec, body = doJSON(engine, http.MethodGet, "/api/v1/organizations/current/members", "", authHeaders)
	if rec.Code != http.StatusOK {
		t.Fatalf("members status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(body, "buyer@example.com") {
		t.Errorf("members response should list the owner: %s", body)
	}

	// Login issues a fresh session cookie.
	rec, body = doJSON(engine, http.MethodPost, "/api/v1/auth/login", loginBody, csrfHeaders())
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", rec.Code, http.StatusOK)
	}
	if strings.Contains(body, "tok-") {
		t.Error("raw session token must never appear in a response body")
	}
	loginCookie := sessionCookieFrom(t, rec)

	// Logout revokes the session server-side and clears the cookie.
	logoutHeaders := csrfHeaders()
	logoutHeaders["Cookie"] = handler.SessionCookieName + "=" + loginCookie
	rec, _ = doJSON(engine, http.MethodPost, "/api/v1/auth/logout", "", logoutHeaders)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	// The logged-out session no longer authorizes requests.
	rec, body = doJSON(engine, http.MethodGet, "/api/v1/me", "",
		map[string]string{"Cookie": handler.SessionCookieName + "=" + loginCookie})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("me after logout status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLoginWithUnknownCredentialsIsGeneric401(t *testing.T) {
	engine := New(newTestDeps(nil, 10))

	rec, body := doJSON(engine, http.MethodPost, "/api/v1/auth/login",
		`{"email":"ghost@example.com","password":"whatever password"}`, csrfHeaders())
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(body, `"code":"invalid_credentials"`) {
		t.Errorf("unexpected body: %s", body)
	}
	if strings.Contains(body, "ghost@example.com") {
		t.Error("error must not echo identifying details")
	}
}

func TestAuthEndpointsAreRateLimitedPerIP(t *testing.T) {
	engine := New(newTestDeps(nil, 2))

	for i := 0; i < 2; i++ {
		rec, _ := doJSON(engine, http.MethodPost, "/api/v1/auth/login",
			`{"email":"ghost@example.com","password":"wrong password 1"}`, csrfHeaders())
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want %d", i+1, rec.Code, http.StatusUnauthorized)
		}
	}

	rec, body := doJSON(engine, http.MethodPost, "/api/v1/auth/login",
		`{"email":"ghost@example.com","password":"wrong password 2"}`, csrfHeaders())
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d (rate limited)", rec.Code, http.StatusTooManyRequests)
	}
	if !strings.Contains(body, `"code":"rate_limited"`) {
		t.Errorf("unexpected body: %s", body)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("429 responses should include Retry-After")
	}
}

// --- Transport safety net (Milestone 1 behavior, kept) ---

func TestUnknownRouteReturnsErrorEnvelope(t *testing.T) {
	engine := New(newTestDeps(nil, 10))

	rec, body := do(engine, http.MethodGet, "/does-not-exist")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var envelope handler.ErrorEnvelope
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("body is not the error envelope: %v\nbody: %s", err, body)
	}
	if envelope.Error.Code != "not_found" {
		t.Errorf("code = %q, want %q", envelope.Error.Code, "not_found")
	}
	if envelope.Error.RequestID == "" {
		t.Error("error envelope should include a request_id")
	}
	if rec.Header().Get(headerRequestID) == "" {
		t.Error("response should echo the X-Request-Id header")
	}
}

func TestMethodNotAllowedReturnsErrorEnvelope(t *testing.T) {
	engine := New(newTestDeps(nil, 10))

	rec, body := do(engine, http.MethodPost, "/healthz")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if !strings.Contains(body, `"code":"method_not_allowed"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestPanicReturnsSanitizedInternalError(t *testing.T) {
	engine := New(newTestDeps(nil, 10))
	engine.GET("/boom", func(c *gin.Context) { panic("secret internal detail") })

	rec, body := do(engine, http.MethodGet, "/boom")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(body, `"code":"internal_error"`) {
		t.Errorf("unexpected body: %s", body)
	}
	if strings.Contains(body, "secret internal detail") {
		t.Error("panic detail must not leak to the client response")
	}
}

func TestRequestIDHeaderIsEchoedWhenProvided(t *testing.T) {
	engine := New(newTestDeps(nil, 10))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(headerRequestID, "caller-supplied-id")
	engine.ServeHTTP(rec, req)

	if got := rec.Header().Get(headerRequestID); got != "caller-supplied-id" {
		t.Errorf("X-Request-Id = %q, want %q", got, "caller-supplied-id")
	}
}
