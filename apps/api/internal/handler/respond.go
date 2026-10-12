package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorBody is the machine-readable part of an error response.
type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// ErrorEnvelope is the single error response shape used by every
// endpoint so clients can parse errors consistently.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// WriteError emits the standard error envelope. Messages must be
// sanitized for clients; details belong in server-side logs only.
// Exported because the server layer (no-route handlers, middleware)
// shares the exact same shape.
func WriteError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorEnvelope{Error: ErrorBody{
		Code:      code,
		Message:   message,
		RequestID: RequestIDFrom(c),
	}})
}

// writeUnexpectedError logs an unexpected service/store error with its
// operation and answers with the sanitized 500 envelope.
func writeUnexpectedError(logger *slog.Logger, c *gin.Context, operation string, err error) {
	logger.Error("unexpected service error",
		slog.String("operation", operation),
		slog.String("error", err.Error()),
	)
	WriteError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
}
