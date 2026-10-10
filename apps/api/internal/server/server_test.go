package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/madhavr16/QuoteWise/apps/api/internal/config"
	"github.com/madhavr16/QuoteWise/apps/api/internal/handler"
	"github.com/madhavr16/QuoteWise/apps/api/internal/service"
	"github.com/madhavr16/QuoteWise/apps/api/internal/version"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

type fakePinger struct {
	err error
}

func (f fakePinger) Ping(context.Context) error { return f.err }

// The server layer is tested end-to-end through real handlers and a
// real service instance wired to fake ports (the same way cmd/api
// wires the production graph).
func newTestDeps(checks map[string]service.Pinger) Deps {
	return Deps{
		Config: config.Config{
			Environment:  "development",
			HTTPPort:     8080,
			LogLevel:     "info",
			LogFormat:    "json",
			DatabaseURL:  "postgresql://user:pass@localhost:5432/quotewise",
			AIServiceURL: "http://localhost:8000",
		},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Handlers: Handlers{
			Health: handler.NewHealthHandler(service.NewHealthService(checks)),
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

func TestHealthzIsAlwaysOK(t *testing.T) {
	engine := New(newTestDeps(nil))

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
	}))

	rec, body := do(engine, http.MethodGet, "/readyz")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(body, `"status":"ready"`) {
		t.Errorf("unexpected body: %s", body)
	}
	if !strings.Contains(body, `"postgres":{"status":"ok"}`) {
		t.Errorf("expected postgres check ok in body: %s", body)
	}
	if !strings.Contains(body, `"ai_service":{"status":"ok"}`) {
		t.Errorf("expected ai_service check ok in body: %s", body)
	}
}

func TestReadyzFailsWhenADependencyFails(t *testing.T) {
	engine := New(newTestDeps(map[string]service.Pinger{
		"postgres":   fakePinger{err: errors.New("connection refused")},
		"ai_service": fakePinger{},
	}))

	rec, body := do(engine, http.MethodGet, "/readyz")
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(body, `"status":"not_ready"`) {
		t.Errorf("unexpected body: %s", body)
	}
	if !strings.Contains(body, "connection refused") {
		t.Errorf("expected failure detail in body: %s", body)
	}
}

func TestUnknownRouteReturnsErrorEnvelope(t *testing.T) {
	engine := New(newTestDeps(nil))

	rec, body := do(engine, http.MethodGet, "/does-not-exist")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var envelope ErrorEnvelope
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
	engine := New(newTestDeps(nil))

	rec, body := do(engine, http.MethodPost, "/healthz")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if !strings.Contains(body, `"code":"method_not_allowed"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestPanicReturnsSanitizedInternalError(t *testing.T) {
	engine := New(newTestDeps(nil))
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
	engine := New(newTestDeps(nil))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(headerRequestID, "caller-supplied-id")
	engine.ServeHTTP(rec, req)

	if got := rec.Header().Get(headerRequestID); got != "caller-supplied-id" {
		t.Errorf("X-Request-Id = %q, want %q", got, "caller-supplied-id")
	}
}
