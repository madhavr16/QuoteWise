package domain

import (
	"testing"
	"time"
)

func TestSessionExpired(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	session := Session{ExpiresAt: now.Add(time.Hour)}

	if session.Expired(now) {
		t.Error("session that expires in the future should not be expired")
	}
	if !session.Expired(now.Add(time.Hour)) {
		t.Error("session should be expired at its expiry time")
	}
	if !session.Expired(now.Add(2 * time.Hour)) {
		t.Error("session should be expired after its expiry time")
	}
}
