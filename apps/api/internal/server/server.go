// Package server assembles the HTTP transport: the Gin engine, global
// middleware, the error envelope, and route registration. It depends
// on handlers (which depend on services); it never touches stores,
// gateways, or SQL directly (ADR-015).
package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/madhavr16/QuoteWise/apps/api/internal/config"
	"github.com/madhavr16/QuoteWise/apps/api/internal/handler"
	"github.com/madhavr16/QuoteWise/apps/api/internal/service"
)

// Deps bundles the server's dependencies.
type Deps struct {
	Config   config.Config
	Logger   *slog.Logger
	Handlers Handlers
	// Auth is the auth use case; the Sessions middleware resolves
	// cookies through it.
	Auth *service.AuthService
}

// Handlers carries every mounted handler; one field per resource.
type Handlers struct {
	Health       *handler.HealthHandler
	Auth         *handler.AuthHandler
	Org          *handler.OrgHandler
	Requests     *handler.ProcurementRequestHandler
	Quotations   *handler.QuotationUploadHandler
}

// New builds the Gin engine with global middleware and routes.
func New(deps Deps) *gin.Engine {
	if deps.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(RequestID(), RequestLogger(deps.Logger), Recovery(deps.Logger))

	// Infrastructure endpoints (deliberately unversioned).
	r.GET("/healthz", deps.Handlers.Health.Healthz)
	r.GET("/readyz", deps.Handlers.Health.Readyz)

	// Business API (Milestone 2 part 1: authentication and
	// organization management; procurement resources land with
	// Milestone 2 part 2).
	api := r.Group("/api/v1")
	api.Use(
		Sessions(deps.Auth, deps.Logger),
		RequireCSRFForMutations(),
	)

	// Auth endpoints carry the per-IP rate limit (spec §11).
	auth := api.Group("/auth", AuthRateLimit(NewRateLimiter(
		deps.Config.AuthRateLimitPerMinute, time.Minute,
	)))
	auth.POST("/register", deps.Handlers.Auth.Register)
	auth.POST("/login", deps.Handlers.Auth.Login)
	// Logout stays idempotent without a session, so it is not behind
	// RequireAuth.
	auth.POST("/logout", deps.Handlers.Auth.Logout)

	// Everything else under /api/v1 requires a session.
	protected := api.Group("")
	protected.Use(RequireAuth())
	protected.GET("/me", deps.Handlers.Org.Me)
	protected.GET("/organizations/current", deps.Handlers.Org.CurrentOrganization)
	protected.GET("/organizations/current/members", deps.Handlers.Org.Members)

	// Procurement requests and their quotations (Milestone 2 part 2).
	protected.POST("/procurement-requests", deps.Handlers.Requests.Create)
	protected.GET("/procurement-requests", deps.Handlers.Requests.List)
	protected.GET("/procurement-requests/:id", deps.Handlers.Requests.Get)
	protected.POST("/procurement-requests/:id/quotations", deps.Handlers.Quotations.Upload)
	protected.GET("/procurement-requests/:id/quotations", deps.Handlers.Quotations.List)

	r.HandleMethodNotAllowed = true
	r.NoMethod(func(c *gin.Context) {
		handler.WriteError(c, http.StatusMethodNotAllowed, "method_not_allowed",
			"The HTTP method is not allowed for this resource.")
	})
	r.NoRoute(func(c *gin.Context) {
		handler.WriteError(c, http.StatusNotFound, "not_found",
			"The requested resource was not found.")
	})

	return r
}
