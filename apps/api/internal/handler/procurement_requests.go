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

// ProcurementRequestHandler serves the procurement-request endpoints
// (Milestone 2 part 2). Handlers stay thin: bind JSON, resolve the
// auth context, call the service, answer with DTOs (ADR-015).
type ProcurementRequestHandler struct {
	requests *service.ProcurementRequestService
	logger   *slog.Logger
}

// NewProcurementRequestHandler wires the procurement-request
// endpoints.
func NewProcurementRequestHandler(requests *service.ProcurementRequestService, logger *slog.Logger) *ProcurementRequestHandler {
	return &ProcurementRequestHandler{requests: requests, logger: logger}
}

type procurementRequestResponse struct {
	ID              string               `json:"id"`
	Title           string               `json:"title"`
	Status          domain.RequestStatus `json:"status"`
	Currency        string               `json:"currency"`
	RequesterUserID string               `json:"requester_user_id"`
	CreatedAt       string               `json:"created_at"`
	UpdatedAt       string               `json:"updated_at"`
}

type procurementRequestsResponse struct {
	Requests []procurementRequestResponse `json:"requests"`
}

// respondRequestError maps expected service errors to envelopes; it
// returns true when the error was expected (and answered).
func (h *ProcurementRequestHandler) respondRequestError(c *gin.Context, err error) bool {
	var validation *service.ValidationError
	switch {
	case errors.As(err, &validation):
		WriteError(c, http.StatusUnprocessableEntity, "validation_error", validation.Error())
	case errors.Is(err, domain.ErrNotFound):
		WriteError(c, http.StatusNotFound, "not_found", "The requested resource was not found.")
	default:
		return false
	}
	return true
}

// Create creates a draft procurement request under the signed-in
// user's organization.
func (h *ProcurementRequestHandler) Create(c *gin.Context) {
	authCtx, ok := authOrReject(c)
	if !ok {
		return
	}

	var req service.CreateProcurementRequestInput
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, http.StatusBadRequest, "bad_request", "Request body must be valid JSON.")
		return
	}

	request, err := h.requests.CreateProcurementRequest(c.Request.Context(), authCtx, req)
	if err != nil {
		if !h.respondRequestError(c, err) {
			writeUnexpectedError(h.logger, c, "create request", err)
		}
		return
	}
	c.JSON(http.StatusCreated, toProcurementRequestResponse(request))
}

// List returns the organization's requests, newest first.
func (h *ProcurementRequestHandler) List(c *gin.Context) {
	authCtx, ok := authOrReject(c)
	if !ok {
		return
	}

	requests, err := h.requests.RequestsByOrganization(c.Request.Context(), authCtx.Organization.ID)
	if err != nil {
		writeUnexpectedError(h.logger, c, "list requests", err)
		return
	}

	responses := make([]procurementRequestResponse, 0, len(requests))
	for _, request := range requests {
		responses = append(responses, toProcurementRequestResponse(request))
	}
	c.JSON(http.StatusOK, procurementRequestsResponse{Requests: responses})
}

// Get returns one request by ID.
func (h *ProcurementRequestHandler) Get(c *gin.Context) {
	if _, ok := authOrReject(c); !ok {
		return
	}

	request, err := h.requests.RequestByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if !h.respondRequestError(c, err) {
			writeUnexpectedError(h.logger, c, "get request", err)
		}
		return
	}
	c.JSON(http.StatusOK, toProcurementRequestResponse(request))
}

func toProcurementRequestResponse(request domain.ProcurementRequest) procurementRequestResponse {
	return procurementRequestResponse{
		ID:              request.ID,
		Title:           request.Title,
		Status:          request.Status,
		Currency:        request.Currency,
		RequesterUserID: request.RequesterUserID,
		CreatedAt:       request.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:       request.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}