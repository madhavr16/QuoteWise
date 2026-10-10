package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

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

// The handler is tested through a real service instance with fake
// ports — no HTTP mocking of the service layer is needed.
func newTestRouter(deps map[string]service.Pinger) *gin.Engine {
	r := gin.New()
	h := NewHealthHandler(service.NewHealthService(deps))
	r.GET("/healthz", h.Healthz)
	r.GET("/readyz", h.Readyz)
	return r
}

func do(engine *gin.Engine, method, path string) (*httptest.ResponseRecorder, string) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	engine.ServeHTTP(rec, req)
	body, _ := io.ReadAll(rec.Body)
	return rec, string(body)
}

func TestHealthzIsAlwaysOK(t *testing.T) {
	router := newTestRouter(nil)

	rec, body := do(router, http.MethodGet, "/healthz")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(body, `"status":"ok"`) || !strings.Contains(body, version.Version) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestReadyzPassesWhenAllDependenciesPass(t *testing.T) {
	router := newTestRouter(map[string]service.Pinger{
		"postgres":   fakePinger{},
		"ai_service": fakePinger{},
	})

	rec, body := do(router, http.MethodGet, "/readyz")
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
	router := newTestRouter(map[string]service.Pinger{
		"postgres":   fakePinger{err: errors.New("connection refused")},
		"ai_service": fakePinger{},
	})

	rec, body := do(router, http.MethodGet, "/readyz")
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
