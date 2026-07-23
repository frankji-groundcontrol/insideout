package auth

import (
	"testing"
	"time"

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
