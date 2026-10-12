package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// tokenBytes is 32 bytes → 256 bits of entropy → 43 base64url
// characters; plenty for opaque session tokens (ADR-007).
const tokenBytes = 32

// RandomTokenGenerator implements the service layer's TokenGenerator
// port with crypto/rand.
type RandomTokenGenerator struct{}

// NewRandomTokenGenerator returns the crypto/rand token generator.
func NewRandomTokenGenerator() *RandomTokenGenerator {
	return &RandomTokenGenerator{}
}

// Generate returns a fresh unpredictable opaque token.
func (RandomTokenGenerator) Generate() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
