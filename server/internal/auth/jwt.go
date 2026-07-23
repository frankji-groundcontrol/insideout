package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenIssuer struct {
	secret    []byte
	accessTTL time.Duration
}

func NewTokenIssuer(secret string, accessTTL time.Duration) *TokenIssuer {
	return &TokenIssuer{secret: []byte(secret), accessTTL: accessTTL}
}

// MintAccessToken issues a short-lived HS256 JWT with sub/exp/iat claims.
func (t *TokenIssuer) MintAccessToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(t.accessTTL)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(t.secret)
	if err != nil {
		return "", fmt.Errorf("auth: sign access token: %w", err)
	}
	return signed, nil
}

// VerifyAccessToken parses and validates an access token, returning the
// subject user id.
func (t *TokenIssuer) VerifyAccessToken(raw string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(tok *jwt.Token) (interface{}, error) {
		if _, ok := tok.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", tok.Header["alg"])
		}
		return t.secret, nil
	})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("auth: parse access token: %w", err)
	}
	if !token.Valid {
		return uuid.UUID{}, fmt.Errorf("auth: invalid access token")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("auth: invalid subject claim: %w", err)
	}
	return userID, nil
}
