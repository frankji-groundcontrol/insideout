package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// GenerateRefreshToken returns a 256-bit random opaque token (URL-safe
// base64) plus its sha256 hex digest for storage. Only the digest is ever
// persisted in insideout.sessions.
func GenerateRefreshToken() (token string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("auth: generate refresh token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(buf)
	return token, HashRefreshToken(token), nil
}

func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
