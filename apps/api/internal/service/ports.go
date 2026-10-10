package service

import "context"

// Pinger is a port implemented by infrastructure adapters (for example
// store/postgres or gateway/aiservice). Services depend on this
// interface — never on concrete drivers — so tests use fakes and
// wiring stays swappable. Go interfaces are satisfied implicitly, so
// adapters do not import this package to implement it.
type Pinger interface {
	Ping(ctx context.Context) error
}
