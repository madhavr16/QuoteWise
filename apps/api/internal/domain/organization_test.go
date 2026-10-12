package domain

import (
	"strings"
	"testing"
)

func TestValidateOrganizationName(t *testing.T) {
	if err := ValidateOrganizationName("  Acme Procurement  "); err != nil {
		t.Errorf("valid name rejected: %v", err)
	}
	if err := ValidateOrganizationName("   "); err == nil {
		t.Error("blank name should be rejected")
	}
	if err := ValidateOrganizationName(strings.Repeat("n", 201)); err == nil {
		t.Error("name longer than 200 characters should be rejected")
	}
	if err := ValidateOrganizationName(strings.Repeat("n", 200)); err != nil {
		t.Errorf("200-character name should be accepted: %v", err)
	}
}
