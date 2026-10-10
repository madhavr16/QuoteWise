// Package aiservice is the outbound adapter (gateway) to the internal
// Python AI service. It implements the service layer's ports over
// HTTP — implicitly, via Go structural typing, so this package never
// imports internal/service. Document extraction, embeddings, and
// retrieval calls land here in Milestone 3+ (see
// docs/implementation-plan.md).
package aiservice

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Gateway calls the internal AI service over HTTP. The service is
// internal-only (never exposed to browsers); every call carries a
// timeout in addition to the caller's deadline.
type Gateway struct {
	baseURL string
	client  *http.Client
}

// New builds a gateway for the given base URL (for example
// http://ai-service:8000 inside the Compose network).
func New(baseURL string) *Gateway {
	return &Gateway{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// Ping reports AI-service reachability (satisfies the service layer's
// Pinger port); any 2xx response counts as healthy.
func (g *Gateway) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("ai service returned status %d", resp.StatusCode)
	}
	return nil
}
