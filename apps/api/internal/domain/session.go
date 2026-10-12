package domain

import "time"

// Session is a server-side login session (ADR-007). TokenHash is the
// hex-encoded SHA-256 of the opaque token delivered in the cookie; the
// raw token is never stored, so a database leak does not expose usable
// sessions.
type Session struct {
	ID        string
	UserID    string
	TokenHash string
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
}

// Expired reports whether the session is no longer valid at time t.
func (s Session) Expired(t time.Time) bool {
	return !t.Before(s.ExpiresAt)
}
