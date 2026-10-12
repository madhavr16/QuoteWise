package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// User is an account that can sign in (spec §6). Email is stored
// normalized; PasswordHash is an opaque PHC-encoded string produced by
// the security adapter (argon2id).
type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// emailPattern is a pragmatic shape check, not a full RFC 5322
// implementation — deliverability is out of MVP scope (no email is
// ever sent), so shape plus length is the right level of checking.
var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// NormalizeEmail trims and lowercases an address and validates its
// shape. The normalized form is the only form ever stored or compared.
func NormalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	switch {
	case normalized == "":
		return "", fmt.Errorf("email is required")
	case len(normalized) > 320:
		return "", fmt.Errorf("email must be at most 320 characters")
	case !emailPattern.MatchString(normalized):
		return "", fmt.Errorf("email must be a valid address")
	}
	return normalized, nil
}

// ValidatePassword enforces the password policy: length only, no
// composition rules (NIST SP 800-63B style). 12 characters is the
// minimum for a business procurement tool; 128 bounds the argon2 work
// per verification attempt.
func ValidatePassword(password string) error {
	switch {
	case len(password) < 12:
		return fmt.Errorf("password must be at least 12 characters")
	case len(password) > 128:
		return fmt.Errorf("password must be at most 128 characters")
	default:
		return nil
	}
}
