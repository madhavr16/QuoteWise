package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
)

// ProcurementRequestService implements the procurement-request use
// cases (Milestone 2 part 2): create a draft request under the
// signed-in user's organization and list/view the organization's
// requests. Every operation is scoped to the organization resolved
// from the session — never from client input (spec §11).
type ProcurementRequestService struct {
	requests ProcurementRequestStore
	logger   *slog.Logger
}

// NewProcurementRequestService wires the procurement-request use
// case. The store satisfies ProcurementRequestStore via structural
// typing (ADR-015).
func NewProcurementRequestService(requests ProcurementRequestStore, logger *slog.Logger) *ProcurementRequestService {
	return &ProcurementRequestService{requests: requests, logger: logger}
}

// CreateProcurementRequest validates input through domain rules and
// persists a draft request for the organization.
func (s *ProcurementRequestService) CreateProcurementRequest(ctx context.Context, authCtx AuthContext, input CreateProcurementRequestInput) (domain.ProcurementRequest, error) {
	var fields []FieldError

	title := strings.TrimSpace(input.Title)
	if err := domain.ValidateRequestTitle(title); err != nil {
		fields = append(fields, FieldError{Field: "title", Message: err.Error()})
	}
	currency := strings.TrimSpace(strings.ToUpper(input.Currency))
	if err := domain.ValidateCurrency(currency); err != nil {
		fields = append(fields, FieldError{Field: "currency", Message: err.Error()})
	}
	if len(fields) > 0 {
		return domain.ProcurementRequest{}, &ValidationError{Fields: fields}
	}

	created, err := s.requests.CreateProcurementRequest(ctx, domain.ProcurementRequest{
		OrganizationID:  authCtx.Organization.ID,
		RequesterUserID: authCtx.User.ID,
		Title:           title,
		Currency:        currency,
		Status:          domain.RequestDraft,
	})
	if err != nil {
		s.logger.Error("create procurement request", slog.Any("error", err))
		return domain.ProcurementRequest{}, err
	}
	return created, nil
}

// RequestsByOrganization lists the organization's requests, newest
// first.
func (s *ProcurementRequestService) RequestsByOrganization(ctx context.Context, organizationID string) ([]domain.ProcurementRequest, error) {
	return s.requests.ProcurementRequestsByOrganization(ctx, organizationID)
}

// RequestByID returns one request.
func (s *ProcurementRequestService) RequestByID(ctx context.Context, id string) (domain.ProcurementRequest, error) {
	return s.requests.ProcurementRequestByID(ctx, id)
}

// CreateProcurementRequestInput is the handler-level payload for
// creating a request. It lives here (not in domain) because it is a
// transport shape; the domain invariant rules are validated from it.
type CreateProcurementRequestInput struct {
	Title    string `json:"title"`
	Currency string `json:"currency"`
}