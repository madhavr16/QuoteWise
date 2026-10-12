package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/madhavr16/QuoteWise/apps/api/internal/handler"
)

// rateLimiterPurgeAt caps tracked keys before a lazy purge of expired
// windows; without it a flood of spoofed IPs would grow the map
// unboundedly.
const rateLimiterPurgeAt = 10_000

// RateLimiter is an approximate in-memory fixed-window limiter for
// auth endpoints (spec §11). Deliberately simple: it is single-
// instance only and a Redis-backed limiter replaces it when the app
// runs more than one instance (documented in decisions.md ADR-017).
type RateLimiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	hits   map[string]*windowHits
}

type windowHits struct {
	count   int
	expires time.Time
}

// NewRateLimiter allows max attempts per key per window.
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		max:    max,
		window: window,
		hits:   make(map[string]*windowHits),
	}
}

// Allow reports whether the key may proceed at the given time. The
// time parameter keeps the limiter unit-testable without sleeps.
func (l *RateLimiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.hits) >= rateLimiterPurgeAt {
		for k, hits := range l.hits {
			if now.After(hits.expires) {
				delete(l.hits, k)
			}
		}
	}

	hits, ok := l.hits[key]
	if !ok || now.After(hits.expires) {
		l.hits[key] = &windowHits{count: 1, expires: now.Add(l.window)}
		return true
	}
	if hits.count >= l.max {
		return false
	}
	hits.count++
	return true
}

// AuthRateLimit bounds auth-endpoint attempts per client IP.
func AuthRateLimit(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.Allow(c.ClientIP(), time.Now()) {
			c.Header("Retry-After", "60")
			handler.WriteError(c, http.StatusTooManyRequests, "rate_limited",
				"Too many attempts. Please try again shortly.")
			c.Abort()
			return
		}
		c.Next()
	}
}
