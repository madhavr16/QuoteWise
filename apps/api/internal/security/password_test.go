package security

import (
	"strings"
	"testing"
)

func TestArgon2HashAndVerifyRoundtrip(t *testing.T) {
	hasher := NewArgon2Hasher()

	hash, err := hasher.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=65536,t=2,p=2$") {
		t.Errorf("hash is not in the expected PHC format: %s", hash)
	}

	ok, err := hasher.Verify("correct horse battery staple", hash)
	if err != nil || !ok {
		t.Errorf("Verify(correct) = (%v, %v), want (true, nil)", ok, err)
	}

	ok, err = hasher.Verify("wrong horse battery staple", hash)
	if err != nil {
		t.Fatalf("Verify(wrong) errored: %v", err)
	}
	if ok {
		t.Error("Verify(wrong) = true, want false")
	}
}

func TestArgon2HashesAreSalted(t *testing.T) {
	hasher := NewArgon2Hasher()

	first, err := hasher.Hash("same password")
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}
	second, err := hasher.Hash("same password")
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}
	if first == second {
		t.Error("two hashes of the same password must differ (random salt)")
	}
}

func TestVerifyRejectsMalformedHashes(t *testing.T) {
	hasher := NewArgon2Hasher()

	for _, malformed := range []string{
		"",
		"plaintext",
		"$argon2i$v=19$m=65536,t=2,p=2$abc$def",  // wrong variant
		"$argon2id$v=18$m=65536,t=2,p=2$abc$def", // wrong version
		"$argon2id$v=19$m=0,t=2,p=2$AAAA$BBBB",   // zero memory
	} {
		if ok, err := hasher.Verify("password", malformed); ok || err == nil {
			t.Errorf("Verify(password, %q) = (%v, %v), want (false, error)", malformed, ok, err)
		}
	}
}
