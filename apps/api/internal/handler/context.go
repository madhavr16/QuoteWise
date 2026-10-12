package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/madhavr16/QuoteWise/apps/api/internal/service"
)

// Gin context plumbing shared by handlers and the server layer's
// middleware. The server package imports this one — never the other
// way around (ADR-015) — so middleware and handlers agree on a single
// set of context keys.

const (
	ginKeyRequestID   = "handler.request_id"
	ginKeyAuthContext = "handler.auth_context"
)

// WithRequestID stores the request correlation ID set by the server's
// RequestID middleware.
func WithRequestID(c *gin.Context, id string) {
	c.Set(ginKeyRequestID, id)
}

// RequestIDFrom returns the request correlation ID, or "unknown" when
// the RequestID middleware did not run (isolated tests).
func RequestIDFrom(c *gin.Context) string {
	if v, ok := c.Get(ginKeyRequestID); ok {
		if id, ok := v.(string); ok && id != "" {
			return id
		}
	}
	return "unknown"
}

// WithAuthContext stores the resolved session context. The server's
// Sessions middleware calls this after resolving the cookie.
func WithAuthContext(c *gin.Context, authCtx service.AuthContext) {
	c.Set(ginKeyAuthContext, authCtx)
}

// AuthContextFrom returns the resolved session context, if present.
func AuthContextFrom(c *gin.Context) (service.AuthContext, bool) {
	v, ok := c.Get(ginKeyAuthContext)
	if !ok {
		return service.AuthContext{}, false
	}
	authCtx, ok := v.(service.AuthContext)
	return authCtx, ok
}
