package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/madhavr16/QuoteWise/apps/api/internal/service"
)

// OrgHandler serves the "me", current-organization, and member-listing
// endpoints. Every query is scoped to the organization resolved from
// the session — never from client input (spec §11).
type OrgHandler struct {
	organizations *service.OrganizationService
	logger        *slog.Logger
}

// NewOrgHandler wires the organization endpoints.
func NewOrgHandler(organizations *service.OrganizationService, logger *slog.Logger) *OrgHandler {
	return &OrgHandler{organizations: organizations, logger: logger}
}

type meResponse struct {
	User         userResponse         `json:"user"`
	Organization organizationResponse `json:"organization"`
	Role         string               `json:"role"`
}

type memberResponse struct {
	UserID   string    `json:"user_id"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type membersResponse struct {
	Members []memberResponse `json:"members"`
}

func authOrReject(c *gin.Context) (service.AuthContext, bool) {
	authCtx, ok := AuthContextFrom(c)
	if !ok {
		// RequireAuth normally guarantees this; handlers stay
		// defensive so wiring mistakes fail closed, not open.
		WriteError(c, http.StatusUnauthorized, "authentication_required",
			"Sign in to use this resource.")
		return service.AuthContext{}, false
	}
	return authCtx, true
}

// Me returns the signed-in user with their current organization and role.
func (h *OrgHandler) Me(c *gin.Context) {
	authCtx, ok := authOrReject(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, meResponse{
		User:         toUserResponse(authCtx.User),
		Organization: toOrganizationResponse(authCtx.Organization),
		Role:         string(authCtx.Membership.Role),
	})
}

// CurrentOrganization returns the user's current organization.
func (h *OrgHandler) CurrentOrganization(c *gin.Context) {
	authCtx, ok := authOrReject(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, toOrganizationResponse(authCtx.Organization))
}

// Members lists the current organization's members (any member may
// view; inviting members lands with open decision P6).
func (h *OrgHandler) Members(c *gin.Context) {
	authCtx, ok := authOrReject(c)
	if !ok {
		return
	}

	members, err := h.organizations.Members(c.Request.Context(), authCtx.Organization.ID)
	if err != nil {
		writeUnexpectedError(h.logger, c, "list members", err)
		return
	}

	responses := make([]memberResponse, 0, len(members))
	for _, member := range members {
		responses = append(responses, memberResponse{
			UserID:   member.UserID,
			Email:    member.Email,
			Role:     string(member.Role),
			JoinedAt: member.JoinedAt,
		})
	}
	c.JSON(http.StatusOK, membersResponse{Members: responses})
}
