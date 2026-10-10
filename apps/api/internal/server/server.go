// Package server assembles the HTTP transport: the Gin engine, global
// middleware, the error envelope, and route registration. It depends
// on handlers (which depend on services); it never touches stores,
// gateways, or SQL directly. The business API mounts under /api/v1
// from Milestone 2.
package server

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/madhavr16/QuoteWise/apps/api/internal/config"
	"github.com/madhavr16/QuoteWise/apps/api/internal/handler"
)

// Deps bundles the server's dependencies.
type Deps struct {
	Config   config.Config
	Logger   *slog.Logger
	Handlers Handlers
}

// Handlers carries every mounted handler; one field per resource.
type Handlers struct {
	Health *handler.HealthHandler
}

// New builds the Gin engine with global middleware and routes. Route
// groups for the business API (/api/v1) land in later milestones.
func New(deps Deps) *gin.Engine {
	if deps.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(RequestID(), RequestLogger(deps.Logger), Recovery(deps.Logger))

	// Infrastructure endpoints (deliberately unversioned); the
	// business API will live under /api/v1 in later milestones.
	r.GET("/healthz", deps.Handlers.Health.Healthz)
	r.GET("/readyz", deps.Handlers.Health.Readyz)

	r.HandleMethodNotAllowed = true
	r.NoMethod(func(c *gin.Context) {
		writeError(c, http.StatusMethodNotAllowed, "method_not_allowed",
			"The HTTP method is not allowed for this resource.")
	})
	r.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, "not_found",
			"The requested resource was not found.")
	})

	return r
}
