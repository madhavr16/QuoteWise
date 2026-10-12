package domain

import (
	"fmt"
	"strings"
	"time"
)

// Organization is the tenant that owns all organization-scoped data
// (spec §6). The MVP runs a single organization per user (created at
// registration, ADR-007), but the model is multi-organization ready.
type Organization struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ValidateOrganizationName checks a display name after trimming.
func ValidateOrganizationName(name string) error {
	trimmed := strings.TrimSpace(name)
	switch {
	case trimmed == "":
		return fmt.Errorf("organization name is required")
	case len(trimmed) > 200:
		return fmt.Errorf("organization name must be at most 200 characters")
	default:
		return nil
	}
}
