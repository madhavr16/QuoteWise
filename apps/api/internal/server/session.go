package server

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
	"github.com/madhavr16/QuoteWise/apps/api/internal/handler"
	"github.com/madhavr16/QuoteWise/apps/api/internal/service"
)

// Sessions resolves the session cookie into an auth context before
// every /api/v1 request (ADR-007). Anonymous requests continue without
// one; RequireAuth decides what happens next. Resolution failures are
// logged server-side; clients only see a sanitized envelope.
func Sessions(auth *service.AuthService, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(handler.SessionCookieName)
		if err != nil || token == "" {
			c.Next()
			return
		}

		authCtx, err := auth.ResolveSession(c.Request.Context(), token)
		switch {
		case err == nil:
			handler.WithAuthContext(c, authCtx)
		case errors.Is(err, domain.ErrNotFound):
			// No valid session: stay anonymous (401 comes from
			// RequireAuth on protected routes).
		default:
			logger.Error("session resolution failed",
				slog.String("request_id", handler.RequestIDFrom(c)),
				slog.String("error", err.Error()),
			)
			handler.WriteError(c, http.StatusInternalServerError, "internal_error",
				"An unexpected error occurred.")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireAuth rejects anonymous requests with 401.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := handler.AuthContextFrom(c); !ok {
			handler.WriteError(c, http.StatusUnauthorized, "authentication_required",
				"Sign in to use this resource.")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireCSRFForMutations enforces the same-origin custom header on
// mutating methods — the CSRF mitigation layered on top of the
// SameSite=Lax session cookie (ADR-007). The frontend sends it on
// every non-GET request (handler.CSRFHeader).
func RequireCSRFForMutations() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		if c.GetHeader(handler.CSRFHeader) != handler.CSRFHeaderValue {
			handler.WriteError(c, http.StatusForbidden, "csrf_header_required",
				"Mutating requests require the "+handler.CSRFHeader+" header.")
			c.Abort()
			return
		}
		c.Next()
	}
}
