package security

import (
	"strings"
	"testing"
)

func TestTokenGeneratorShape(t *testing.T) {
	token, err := NewRandomTokenGenerator().Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(token) != 43 {
		t.Errorf("token length = %d, want 43 (256-bit base64url)", len(token))
	}
	if strings.ContainsAny(token, "+/=") {
		t.Errorf("token %q contains non-URL-safe characters", token)
	}
}

func TestTokenGeneratorUniqueness(t *testing.T) {
	generator := NewRandomTokenGenerator()
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		token, err := generator.Generate()
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}
		if _, duplicate := seen[token]; duplicate {
			t.Fatalf("duplicate token after %d generations", i)
		}
		seen[token] = struct{}{}
	}
}
