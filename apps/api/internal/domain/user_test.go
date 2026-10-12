package domain

import (
	"strings"
	"testing"
)

func TestNormalizeEmailNormalizesAndAccepts(t *testing.T) {
	cases := []struct {
		name  string
		email string
		want  string
	}{
		{"trims and lowercases", "  User@Example.COM ", "user@example.com"},
		{"plain lowercase", "user@example.com", "user@example.com"},
		{"subdomain", "buyer@procurement.example.co", "buyer@procurement.example.co"},
	}
	for _, tc := range cases {
		got, err := NormalizeEmail(tc.email)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
		if got != tc.want {
			t.Errorf("%s: NormalizeEmail = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestNormalizeEmailRejectsInvalidInput(t *testing.T) {
	invalid := []string{
		"",
		"not-an-email",
		"a@b", // no dot in domain
		"a b@example.com",
		"user@@example.com",
		strings.Repeat("a", 315) + "@x.com", // > 320 chars
	}
	for _, email := range invalid {
		if _, err := NormalizeEmail(email); err == nil {
			t.Errorf("NormalizeEmail(%q) should return an error", email)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("short123"); err == nil {
		t.Error("password shorter than 12 characters should be rejected")
	}
	if err := ValidatePassword(strings.Repeat("x", 129)); err == nil {
		t.Error("password longer than 128 characters should be rejected")
	}
	for _, ok := range []string{"12-char-minimum!", strings.Repeat("x", 128)} {
		if err := ValidatePassword(ok); err != nil {
			t.Errorf("ValidatePassword(%q) = %v, want nil", ok, err)
		}
	}
}
