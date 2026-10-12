// Package security provides the auth-cryptographic adapters: argon2id
// password hashing (ADR-007) and random session-token generation. Pure
// computation — no I/O, no framework imports.
package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters (RFC 9106 recommended profile, ~64 MiB).
// Deliberately constants: they are a security decision, not a tuning
// knob, and every hash carries its own parameters anyway (PHC format),
// so changing these never invalidates stored hashes.
const (
	argon2MemoryKib = 64 * 1024 // 64 MiB
	argon2Time      = 2
	argon2Threads   = 2
	argon2SaltLen   = 16
	argon2KeyLen    = 32
)

// Argon2Hasher implements the service layer's PasswordHasher port with
// argon2id and PHC-encoded hashes
// ($argon2id$v=19$m=...,t=...,p=...$salt$key).
type Argon2Hasher struct{}

// NewArgon2Hasher returns the argon2id hasher with standard parameters.
func NewArgon2Hasher() *Argon2Hasher { return &Argon2Hasher{} }

// Hash derives an argon2id key from the password with a fresh random
// salt and encodes it in PHC format.
func (Argon2Hasher) Hash(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argon2Time, argon2MemoryKib, argon2Threads, argon2KeyLen)
	return encodePHC(argon2Time, argon2MemoryKib, argon2Threads, salt, key), nil
}

// Verify recomputes the key from the password using the parameters
// stored inside the hash and compares in constant time. Malformed
// hashes fail: a corrupt or foreign-format hash must never verify.
func (Argon2Hasher) Verify(password, encodedHash string) (bool, error) {
	params, salt, wantKey, err := decodePHC(encodedHash)
	if err != nil {
		return false, fmt.Errorf("malformed password hash: %w", err)
	}
	gotKey := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(wantKey)))
	return subtle.ConstantTimeCompare(gotKey, wantKey) == 1, nil
}

type phcParams struct {
	time    uint32
	memory  uint32
	threads uint8
}

func encodePHC(time uint32, memory uint32, threads uint8, salt, key []byte) string {
	return "$argon2id$v=19" +
		"$m=" + strconv.FormatUint(uint64(memory), 10) +
		",t=" + strconv.FormatUint(uint64(time), 10) +
		",p=" + strconv.FormatUint(uint64(threads), 10) +
		"$" + base64.RawStdEncoding.EncodeToString(salt) +
		"$" + base64.RawStdEncoding.EncodeToString(key)
}

func decodePHC(encoded string) (phcParams, []byte, []byte, error) {
	// Standard PHC format:
	//   $argon2id$v=19$m=...,t=...,p=...$salt$key
	// splits into ["", "argon2id", "v=19", "m=...", salt, key].
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return phcParams{}, nil, nil, fmt.Errorf("unexpected hash shape")
	}
	if parts[2] != "v=19" {
		return phcParams{}, nil, nil, fmt.Errorf("unsupported argon2 version %q", parts[2])
	}

	var params phcParams
	for _, pair := range strings.Split(parts[3], ",") {
		key, value, found := strings.Cut(pair, "=")
		if !found {
			return phcParams{}, nil, nil, fmt.Errorf("malformed parameter %q", pair)
		}
		switch key {
		case "m":
			parsed, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return phcParams{}, nil, nil, fmt.Errorf("bad m value: %w", err)
			}
			params.memory = uint32(parsed)
		case "t":
			parsed, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return phcParams{}, nil, nil, fmt.Errorf("bad t value: %w", err)
			}
			params.time = uint32(parsed)
		case "p":
			parsed, err := strconv.ParseUint(value, 10, 8)
			if err != nil {
				return phcParams{}, nil, nil, fmt.Errorf("bad p value: %w", err)
			}
			params.threads = uint8(parsed)
		default:
			return phcParams{}, nil, nil, fmt.Errorf("unknown parameter %q", key)
		}
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return phcParams{}, nil, nil, fmt.Errorf("bad salt: %w", err)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return phcParams{}, nil, nil, fmt.Errorf("bad key: %w", err)
	}
	if params.memory == 0 || params.time == 0 || params.threads == 0 || len(salt) == 0 || len(key) == 0 {
		return phcParams{}, nil, nil, fmt.Errorf("incomplete hash parameters")
	}
	return params, salt, key, nil
}
