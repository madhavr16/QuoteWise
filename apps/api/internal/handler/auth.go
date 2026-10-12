package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
	"github.com/madhavr16/QuoteWise/apps/api/internal/service"
)

// Transport contracts shared with the server middleware and the
// frontend (ADR-007).
const (
	// SessionCookieName is the session cookie.
	SessionCookieName = "quotewise_session"
	// CSRFHeader must accompany every mutating request; browsers never
	// add it to a cross-site request without a CORS preflight (CSRF
	// mitigation on top of SameSite=Lax).
	CSRFHeader = "X-Requested-With"
	// CSRFHeaderValue is the required value of CSRFHeader.
	CSRFHeaderValue = "XMLHttpRequest"
)

// AuthHandler serves the authentication endpoints.
type AuthHandler struct {
	auth         *service.AuthService
	cookieSecure bool
	sessionTTL   time.Duration
	logger       *slog.Logger
}

// NewAuthHandler wires the auth endpoints. cookieSecure marks the
// session cookie Secure (production only, where HTTPS terminates
// requests; ADR-007).
func NewAuthHandler(
	auth *service.AuthService,
	cookieSecure bool,
	sessionTTL time.Duration,
	logger *slog.Logger,
) *AuthHandler {
	return &AuthHandler{
		auth:         auth,
		cookieSecure: cookieSecure,
		sessionTTL:   sessionTTL,
		logger:       logger,
	}
}

type registerRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	OrganizationName string `json:"organization_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type organizationResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type authResponse struct {
	User         userResponse          `json:"user"`
	Organization *organizationResponse `json:"organization,omitempty"`
}

// Register creates the user's account, organization, owner
// membership, and first session (ADR-007). The session token is
// delivered only in the HttpOnly cookie — never in a response body.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, http.StatusBadRequest, "bad_request", "Request body must be valid JSON.")
		return
	}

	result, err := h.auth.Register(c.Request.Context(), service.RegisterInput{
		Email:            req.Email,
		Password:         req.Password,
		OrganizationName: req.OrganizationName,
	})
	if err != nil {
		if !h.respondAuthError(c, err) {
			writeUnexpectedError(h.logger, c, "register", err)
		}
		return
	}

	setSessionCookie(c, result.Token, h.cookieSecure, h.sessionTTL)
	c.JSON(http.StatusCreated, authResponse{
		User:         toUserResponse(result.User),
		Organization: toOrganizationResponsePtr(result.Organization),
	})
}

// Login verifies credentials and starts a session.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, http.StatusBadRequest, "bad_request", "Request body must be valid JSON.")
		return
	}

	result, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if !h.respondAuthError(c, err) {
			writeUnexpectedError(h.logger, c, "login", err)
		}
		return
	}

	setSessionCookie(c, result.Token, h.cookieSecure, h.sessionTTL)
	c.JSON(http.StatusOK, authResponse{User: toUserResponse(result.User)})
}

// Logout revokes the current session and clears the cookie. Without a
// session it is an idempotent 204.
func (h *AuthHandler) Logout(c *gin.Context) {
	if token, err := c.Cookie(SessionCookieName); err == nil && token != "" {
		if err := h.auth.Logout(c.Request.Context(), token); err != nil {
			writeUnexpectedError(h.logger, c, "logout", err)
			return
		}
	}
	clearSessionCookie(c, h.cookieSecure)
	c.Status(http.StatusNoContent)
}

// respondAuthError maps expected auth errors to envelopes; it returns
// true when the error was expected (and answered).
func (h *AuthHandler) respondAuthError(c *gin.Context, err error) bool {
	var validation *service.ValidationError
	switch {
	case errors.As(err, &validation):
		WriteError(c, http.StatusUnprocessableEntity, "validation_error", validation.Error())
	case errors.Is(err, domain.ErrEmailTaken):
		// Documented trade-off (ADR-007): register answers 409 so an
		// existing user is pointed at sign-in; login never reveals
		// whether an email exists.
		WriteError(c, http.StatusConflict, "email_taken",
			"An account with this email already exists. Try signing in instead.")
	case errors.Is(err, service.ErrInvalidCredentials):
		WriteError(c, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password.")
	default:
		return false
	}
	return true
}

func setSessionCookie(c *gin.Context, token string, secure bool, ttl time.Duration) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(SessionCookieName, token, int(ttl.Seconds()), "/", "", secure, true)
}

func clearSessionCookie(c *gin.Context, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(SessionCookieName, "", -1, "/", "", secure, true)
}

func toUserResponse(user domain.User) userResponse {
	return userResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt}
}

func toOrganizationResponse(org domain.Organization) organizationResponse {
	return organizationResponse{ID: org.ID, Name: org.Name, CreatedAt: org.CreatedAt}
}

func toOrganizationResponsePtr(org domain.Organization) *organizationResponse {
	response := toOrganizationResponse(org)
	return &response
}
