package domain

import (
	"fmt"
	"strings"
	"time"
)

// ProcurementRequest is an organization-scoped purchase request.
// One currency per request in the MVP (ADR-006).
type ProcurementRequest struct {
	ID               string
	OrganizationID   string
	RequesterUserID  string
	Title            string
	Status           RequestStatus
	Currency         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// RequestStatus is the procurement request's lifecycle state.
type RequestStatus string

const (
	RequestDraft      RequestStatus = "draft"
	RequestSubmitted  RequestStatus = "submitted"
	RequestProcessing RequestStatus = "processing"
	RequestReady      RequestStatus = "ready"
	RequestFailed     RequestStatus = "failed"
)

// ValidRequestStatus reports whether a status value is known.
func ValidRequestStatus(s RequestStatus) bool {
	switch s {
	case RequestDraft, RequestSubmitted, RequestProcessing, RequestReady, RequestFailed:
		return true
	default:
		return false
	}
}

// ValidateRequestTitle checks a request title after trimming.
func ValidateRequestTitle(title string) error {
	trimmed := strings.TrimSpace(title)
	switch {
	case trimmed == "":
		return fmt.Errorf("title is required")
	case len(trimmed) > 200:
		return fmt.Errorf("title must be at most 200 characters")
	default:
		return nil
	}
}

// ValidateCurrency checks an ISO-4217-style currency code (MVP: 3–10
// uppercase letters; one currency per request, ADR-006).
func ValidateCurrency(currency string) error {
	trimmed := strings.TrimSpace(strings.ToUpper(currency))
	switch {
	case trimmed == "":
		return fmt.Errorf("currency is required")
	case len(trimmed) < 3 || len(trimmed) > 10:
		return fmt.Errorf("currency must be 3 to 10 characters")
	default:
		return nil
	}
}