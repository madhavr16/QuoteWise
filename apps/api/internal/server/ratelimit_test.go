package server

import (
	"testing"
	"time"
)

func TestRateLimiterFixedWindow(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	now := time.Now()

	if !limiter.Allow("ip-1", now) || !limiter.Allow("ip-1", now) {
		t.Fatal("the first two attempts must be allowed")
	}
	if limiter.Allow("ip-1", now) {
		t.Error("the third attempt in the same window must be rejected")
	}

	// Independent windows per key.
	if !limiter.Allow("ip-2", now) {
		t.Error("another client must not be affected")
	}

	// The window resets.
	if !limiter.Allow("ip-1", now.Add(time.Minute+time.Second)) {
		t.Error("attempts must be allowed again after the window expires")
	}
}
