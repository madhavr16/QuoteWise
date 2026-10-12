package domain

// CreateOwnerAccountInput carries the data for atomically creating a
// user, their organization, and the owner membership (registration,
// ADR-007). It lives in the domain (the shared kernel) so persistence
// adapters can implement the owner-account port without importing the
// service layer (ADR-015).
type CreateOwnerAccountInput struct {
	// Email is already normalized (NormalizeEmail).
	Email string
	// PasswordHash is a PHC-encoded argon2id hash.
	PasswordHash string
	// OrganizationName is already trimmed.
	OrganizationName string
}
