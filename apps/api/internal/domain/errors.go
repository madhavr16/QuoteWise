package domain

import "errors"

// Shared-kernel sentinel errors. The domain package is the one
// package every layer may import (ADR-015), so cross-layer sentinels
// live here — adapters (store, gateway) return them, services and
// handlers check them.

// ErrNotFound means a requested record does not exist (or, for
// sessions, that the session is absent or expired).
var ErrNotFound = errors.New("not found")

// ErrEmailTaken means an account with the email already exists.
// Deliberately blunt: it does not describe the existing account.
var ErrEmailTaken = errors.New("email already registered")
