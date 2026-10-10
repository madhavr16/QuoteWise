package server

import (
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

// writeError emits the standard error envelope. Messages must be
// sanitized for clients; details belong in server-side logs only.
func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorEnvelope{Error: ErrorBody{
		Code:      code,
		Message:   message,
		RequestID: requestIDFrom(c),
	}})
}
