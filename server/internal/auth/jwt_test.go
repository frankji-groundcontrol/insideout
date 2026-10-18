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

// claimsOf reads a token's payload the way any client could: base64 JSON, no
// signature check. It pins the wire format of the scope claim.
func claimsOf(t *testing.T, raw string) jwt.MapClaims {
	t.Helper()
	claims := jwt.MapClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(raw, claims); err != nil {
		t.Fatalf("parse claims: %v", err)
	}
	return claims
}

func TestMintAgentTokenCarriesTheAgentScopeAndTheAccessLifetime(t *testing.T) {
	issuer := NewTokenIssuer(testSecret, 90*time.Second)
	userID := uuid.New()
	before := time.Now()

	token, expires, err := issuer.MintAgentToken(userID)
	if err != nil {
		t.Fatalf("MintAgentToken: %v", err)
	}
	gotID, scope, err := issuer.VerifyToken(token)
	if err != nil || gotID != userID || scope != ScopeAgent {
		t.Fatalf("VerifyToken = %v, %q, %v; want the minting user and scope %q", gotID, scope, err, ScopeAgent)
	}
	claims := claimsOf(t, token)
	if claims["scope"] != "agent" {
		t.Errorf("scope claim = %v, want \"agent\"", claims["scope"])
	}
	if exp, _ := claims["exp"].(float64); int64(exp) != expires.Unix() {
		t.Errorf("exp claim = %v, returned expiry = %d; they must agree", claims["exp"], expires.Unix())
	}
	if want := before.Add(90 * time.Second); expires.Before(want) || expires.After(want.Add(5*time.Second)) {
		t.Errorf("expiry = %s, want about %s (the access lifetime, no longer)", expires, want)
	}
	if _, err := issuer.VerifyAccessToken(token); err == nil {
		t.Error("VerifyAccessToken accepted a freshly minted agent token as a full session")
	}
}

func TestAFullSessionCarriesNoScopeClaim(t *testing.T) {
	issuer := NewTokenIssuer(testSecret, time.Minute)
	userID := uuid.New()
	token, err := issuer.MintAccessToken(userID)
	if err != nil {
		t.Fatalf("MintAccessToken: %v", err)
	}
	if _, has := claimsOf(t, token)["scope"]; has {
		t.Error("a full session token has a scope claim; its wire format must stay what every older server and client expects")
	}
	if gotID, scope, err := issuer.VerifyToken(token); err != nil || gotID != userID || scope != ScopeFull {
		t.Errorf("VerifyToken = %v, %q, %v; want the user and the full scope", gotID, scope, err)
	}
	// A token from before scopes existed is the same shape: it must keep working.
	legacy := signWithScope(t, userID, "")
	if _, scope, err := issuer.VerifyToken(legacy); err != nil || scope != ScopeFull {
		t.Errorf("VerifyToken on a scope-less token = %q, %v; want full", scope, err)
	}
}

func TestVerifyTokenRefusesAnUnknownScope(t *testing.T) {
	issuer := NewTokenIssuer(testSecret, time.Minute)
	for _, scope := range []string{"superuser", "AGENT", "full", "agent "} {
		if _, _, err := issuer.VerifyToken(signWithScope(t, uuid.New(), scope)); err == nil {
			t.Errorf("VerifyToken accepted the unknown scope %q", scope)
		}
	}
}
