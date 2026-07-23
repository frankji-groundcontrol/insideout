// Package auth implements password hashing, JWT access tokens, and
// opaque refresh tokens for InsideOut's own email+password auth.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// Argon2id parameters per docs/plans/2026-07-20-go-rewrite/02-backend-go.md §2:
// 64 MiB memory, 1 iteration, 4 parallelism — the x/crypto documented
// recommendation for interactive login.
const (
	argonMemoryKiB   = 64 * 1024
	argonIterations  = 1
	argonParallelism = 4
	argonSaltLen     = 16
	argonKeyLen      = 32
)

// HashPassword returns a PHC-formatted argon2id hash string, e.g.
// "$argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>".
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argonIterations, argonMemoryKiB, argonParallelism, argonKeyLen)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemoryKiB, argonIterations, argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// IsBcryptHash reports whether encoded is a bcrypt hash (Supabase Auth's
// format, $2a$/$2b$/$2y$) rather than our own argon2id format — used by
// the login handler to decide whether a successful verify should trigger
// a transparent upgrade to argon2id (see UpgradeFromBcrypt below). Only
// relevant for the small number of users migrated from the old
// juanleme/auth.users table; every account created through this app's
// own registration endpoint is argon2id from the start.
func IsBcryptHash(encoded string) bool {
	return strings.HasPrefix(encoded, "$2a$") || strings.HasPrefix(encoded, "$2b$") || strings.HasPrefix(encoded, "$2y$")
}

// VerifyPassword checks a plaintext password against a hash produced by
// HashPassword (argon2id) or, for migrated accounts, Supabase Auth's
// bcrypt format — both in constant time (bcrypt's own comparison is
// already constant-time).
func VerifyPassword(encoded, password string) (bool, error) {
	if IsBcryptHash(encoded) {
		err := bcrypt.CompareHashAndPassword([]byte(encoded), []byte(password))
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		return err == nil, err
	}

	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, fmt.Errorf("auth: unrecognized hash format")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("auth: parse version: %w", err)
	}

	var memory uint32
	var iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, fmt.Errorf("auth: parse params: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("auth: decode salt: %w", err)
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("auth: decode hash: %w", err)
	}

	got := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(want)))

	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
