package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateOneTimeToken returns a 256-bit random opaque token (URL-safe
// base64 — safe to drop straight into an emailed query parameter) plus its
// sha256 hex digest for storage. Same construction as the refresh token;
// only the digest is ever persisted, in insideout.auth_tokens. Used for
// emailed links: magic-link login and password reset.
func GenerateOneTimeToken() (token string, hash string, err error) {
	return GenerateRefreshToken()
}

// HashOneTimeToken returns the sha256 hex digest that pairs with
// GenerateOneTimeToken — call it on the raw token from the email link when
// consuming.
func HashOneTimeToken(token string) string {
	return HashRefreshToken(token)
}

// GenerateLoginCode returns a 6-digit numeric sign-in code plus its sha256
// hex digest for storage. crypto/rand.Int is unbiased, unlike a % 1e6 of
// random bytes. The code is short enough to type from an email, so it is
// only ever verified under store.VerifyLoginCode's attempt budget.
func GenerateLoginCode() (code string, hash string, err error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", "", err
	}
	code = fmt.Sprintf("%06d", n.Int64())
	return code, HashOneTimeToken(code), nil
}
