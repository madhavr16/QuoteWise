package server

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/madhavr16/QuoteWise/apps/api/internal/handler"
)

// headerRequestID is the correlation header echoed on every response.
const headerRequestID = "X-Request-Id"

// RequestID assigns every request a correlation ID, honoring an
// inbound X-Request-Id header so callers can correlate their logs.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(headerRequestID)
		if id == "" {
			id = newRequestID()
		}
		handler.WithRequestID(c, id)
		c.Header(headerRequestID, id)
		c.Next()
	}
}

// RequestLogger writes one structured log line per request. It never
// logs request bodies, query strings, or response payloads.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		logger.Info("http_request",
			slog.String("request_id", handler.RequestIDFrom(c)),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		)
	}
}

// Recovery converts panics into sanitized 500 responses; stack traces
// go to server logs only, never to clients.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered",
					slog.String("request_id", handler.RequestIDFrom(c)),
					slog.Any("panic", rec),
					slog.String("stack", string(debug.Stack())),
				)
				handler.WriteError(c, http.StatusInternalServerError, "internal_error",
					"An unexpected error occurred.")
			}
		}()
		c.Next()
	}
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failure is pathological; fall back to a
		// time-based value instead of failing the request.
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b[:])
}
