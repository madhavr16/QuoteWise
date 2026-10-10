// Package handler is the HTTP transport layer: thin request handlers
// that map requests to service calls and service results to response
// bodies. Handlers hold no business logic and touch no infrastructure
// (stores/gateways) directly — see docs/architecture.md §2.2.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/madhavr16/QuoteWise/apps/api/internal/service"
	"github.com/madhavr16/QuoteWise/apps/api/internal/version"
)

// HealthHandler serves the infra-level health endpoints. Liveness is
// transport-level truth (the process is up); readiness delegates to
// the health use case.
type HealthHandler struct {
	health *service.HealthService
}

// NewHealthHandler builds the handler around the health use case.
func NewHealthHandler(health *service.HealthService) *HealthHandler {
	return &HealthHandler{health: health}
}

// Response DTOs (transport models) — deliberately distinct from
// service types so the use case stays transport-agnostic.
type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type checkResult struct {
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type readinessResponse struct {
	Status  string                 `json:"status"`
	Version string                 `json:"version"`
	Checks  map[string]checkResult `json:"checks"`
}

// Healthz reports liveness only. It never touches dependencies so
// probes stay cheap and restart semantics stay correct.
func (h *HealthHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, healthResponse{Status: "ok", Version: version.Version})
}

// Readyz reports readiness by probing dependencies through the health
// use case with a bounded deadline.
func (h *HealthHandler) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	report := h.health.Ready(ctx)

	resp := readinessResponse{
		Version: version.Version,
		Checks:  make(map[string]checkResult, len(report.Checks)),
	}
	for _, check := range report.Checks {
		resp.Checks[check.Name] = checkResult{Status: check.Status, Detail: check.Detail}
	}

	if report.Ready {
		resp.Status = "ready"
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Status = "not_ready"
	c.JSON(http.StatusServiceUnavailable, resp)
}
