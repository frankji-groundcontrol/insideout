package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestTokenIssuer_MintAndVerify_RoundTrip(t *testing.T) {
	issuer := NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Minute)
	userID := uuid.New()

	token, err := issuer.MintAccessToken(userID)
	if err != nil {
		t.Fatalf("MintAccessToken: %v", err)
	}

	got, err := issuer.VerifyAccessToken(token)
	if err != nil {
		t.Fatalf("VerifyAccessToken: %v", err)
	}
	if got != userID {
		t.Fatalf("VerifyAccessToken() = %s, want %s", got, userID)
	}
}

func TestTokenIssuer_VerifyAccessToken_Expired(t *testing.T) {
	issuer := NewTokenIssuer("a-32-byte-or-longer-test-secret!", -time.Minute)
	token, err := issuer.MintAccessToken(uuid.New())
	if err != nil {
		t.Fatalf("MintAccessToken: %v", err)
	}
	if _, err := issuer.VerifyAccessToken(token); err == nil {
		t.Fatal("VerifyAccessToken on an expired token should error, got nil")
	}
}

func TestTokenIssuer_VerifyAccessToken_WrongSecret(t *testing.T) {
	issuer := NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Minute)
	token, err := issuer.MintAccessToken(uuid.New())
	if err != nil {
		t.Fatalf("MintAccessToken: %v", err)
	}

	other := NewTokenIssuer("a-completely-different-secret!!", time.Minute)
	if _, err := other.VerifyAccessToken(token); err == nil {
		t.Fatal("VerifyAccessToken with the wrong secret should error, got nil")
	}
}

// rv4-FEAT-01: a token's scope claim is part of what it may do. VerifyAccessToken
// answers "which user", with no scope in its result, so it must refuse a token
// that carries one: a caller that understands no scopes can then never take an
// agent token (or a scope a later version adds) for a full session.

const testSecret = "a-32-byte-or-longer-test-secret!"

// signWithScope hand-builds a signed access token carrying the scope claim,
// the way a token minted by this server would look.
func signWithScope(t *testing.T, userID uuid.UUID, scope string) string {
	t.Helper()
	now := time.Now()
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID.String(),
		"iat":   now.Unix(),
		"exp":   now.Add(time.Minute).Unix(),
		"scope": scope,
	}).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed
}

func TestVerifyAccessTokenRefusesAnAgentScopedToken(t *testing.T) {
	issuer := NewTokenIssuer(testSecret, time.Minute)
	if _, err := issuer.VerifyAccessToken(signWithScope(t, uuid.New(), "agent")); err == nil {
		t.Fatal("VerifyAccessToken accepted a token with scope \"agent\" as a full session")
	}
}

func TestVerifyAccessTokenRefusesAnUnknownScope(t *testing.T) {
	issuer := NewTokenIssuer(testSecret, time.Minute)
	for _, scope := range []string{"superuser", "AGENT", "full", "agent "} {
		if _, err := issuer.VerifyAccessToken(signWithScope(t, uuid.New(), scope)); err == nil {
			t.Errorf("VerifyAccessToken accepted an unknown scope %q; a scope this server does not know must fail closed", scope)
		}
	}
}
