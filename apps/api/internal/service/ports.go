package service

import (
	"context"
	"time"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
)

// Ports: outbound interfaces defined where they are consumed (this
// layer) and implemented implicitly by adapters in internal/store and
// internal/security (ADR-015). Every parameter and result type is a
// primitive or a domain type — the shared kernel — so adapters never
// import this package. Tests fake these ports instead of using real
// drivers or crypto.

// Pinger reports dependency reachability (health/readiness).
type Pinger interface {
	Ping(ctx context.Context) error
}

// PasswordHasher hashes passwords and verifies candidates. Hashes are
// PHC-encoded strings so parameters travel with the hash.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, encodedHash string) (bool, error)
}

// TokenGenerator produces unpredictable opaque session tokens.
type TokenGenerator interface {
	Generate() (string, error)
}

// OwnerAccountCreator atomically creates a user, their organization,
// and the owner membership — one store transaction (ADR-007).
// Returns domain.ErrEmailTaken when the email is already registered.
type OwnerAccountCreator interface {
	CreateOwnerAccount(ctx context.Context, input domain.CreateOwnerAccountInput) (domain.User, domain.Organization, error)
}

// UserReader reads user records.
type UserReader interface {
	UserByEmail(ctx context.Context, email string) (domain.User, error)
	UserByID(ctx context.Context, id string) (domain.User, error)
}

// MembershipReader reads organization membership. Every query is
// scoped by user or organization identifiers resolved server-side —
// never by client-supplied organization IDs (spec §11).
type MembershipReader interface {
	// MembershipWithOrganizationByUser returns the user's earliest
	// membership with its organization (the single-organization MVP's
	// "current organization"; ADR-007).
	MembershipWithOrganizationByUser(ctx context.Context, userID string) (domain.Membership, domain.Organization, error)
	// MembersByOrganization lists an organization's members.
	MembersByOrganization(ctx context.Context, organizationID string) ([]domain.OrganizationMember, error)
}

// SessionStore persists login sessions. Only token hashes are ever
// stored (ADR-007); raw tokens live in cookies alone.
type SessionStore interface {
	CreateSession(ctx context.Context, session domain.Session) error
	SessionByTokenHash(ctx context.Context, tokenHash string) (domain.Session, error)
	TouchSession(ctx context.Context, id string, expiresAt time.Time) error
	DeleteSession(ctx context.Context, id string) error
	DeleteExpiredSessions(ctx context.Context) error
}

// ProcurementRequestStore persists procurement requests. Every query
// is scoped by organization identifiers resolved server-side — never
// by client-supplied organization IDs (spec §11).
type ProcurementRequestStore interface {
	// CreateProcurementRequest inserts a new request and returns the
	// stored row (server-generated timestamps).
	CreateProcurementRequest(ctx context.Context, request domain.ProcurementRequest) (domain.ProcurementRequest, error)
	// ProcurementRequestsByOrganization lists an organization's
	// requests, newest first.
	ProcurementRequestsByOrganization(ctx context.Context, organizationID string) ([]domain.ProcurementRequest, error)
	// ProcurementRequestByID returns one request. domain.ErrNotFound
	// when absent.
	ProcurementRequestByID(ctx context.Context, id string) (domain.ProcurementRequest, error)
}

// QuotationStore persists quotations. Every query is organization-
// and request-scoped (spec §11).
type QuotationStore interface {
	// CreateQuotation inserts a new quotation row.
	CreateQuotation(ctx context.Context, quotation domain.Quotation) (domain.Quotation, error)
	// QuotationsByRequest lists a request's quotations, newest first.
	QuotationsByRequest(ctx context.Context, requestID string) ([]domain.Quotation, error)
	// QuotationByID returns one quotation. domain.ErrNotFound when
	// absent.
	QuotationByID(ctx context.Context, id string) (domain.Quotation, error)
}

// ProcessingJobStore is the PostgreSQL-backed job queue port (ADR-004).
type ProcessingJobStore interface {
	// CreateProcessingJob enqueues an extraction job for a quotation.
	CreateProcessingJob(ctx context.Context, job domain.ProcessingJob) (domain.ProcessingJob, error)
	// ClaimNextPendingJob atomically claims the oldest pending job
	// (FOR UPDATE SKIP LOCKED) and marks it running.
	ClaimNextPendingJob(ctx context.Context) (domain.ProcessingJob, error)
	// CompleteJob marks a job completed and records its finish time.
	CompleteJob(ctx context.Context, id string) error
	// FailJob marks a job failed with a sanitized error message.
	FailJob(ctx context.Context, id string, errorMessage string) error
}

// StorageUploader persists quotation files to object storage.
// Implementations (MinIO gateway) must return a stable storage key
// that the quotation row references; the raw URL must never be
// constructed by the service or handler (ADR-008).
type StorageUploader interface {
	// UploadBytes stores the given bytes under the implementation's
	// configured bucket and returns the storage key that should be
	// persisted in the quotation row.
	UploadBytes(objectKey string, data []byte) (string, error)
	// Stat reports whether a given object exists in the implementation's
	// configured bucket.
	Stat(objectKey string) (bool, error)
	// Remove deletes an object from the implementation's configured bucket.
	Remove(objectKey string) error
	// Ping reports dependency reachability (health/readiness).
	Ping(ctx context.Context) error
}
