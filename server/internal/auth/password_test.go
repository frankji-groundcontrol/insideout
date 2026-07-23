package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashAndVerifyPassword_RoundTrip(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if got, err := VerifyPassword(hash, "correct-horse-battery-staple"); err != nil || !got {
		t.Fatalf("VerifyPassword(correct) = %v, %v; want true, nil", got, err)
	}
	if got, err := VerifyPassword(hash, "wrong-password"); err != nil || got {
		t.Fatalf("VerifyPassword(wrong) = %v, %v; want false, nil", got, err)
	}
}

func TestHashPassword_DistinctSaltsPerCall(t *testing.T) {
	h1, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	h2, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if h1 == h2 {
		t.Fatalf("two hashes of the same password must differ (random salt), got identical: %s", h1)
	}
}

func TestVerifyPassword_MalformedHash(t *testing.T) {
	if _, err := VerifyPassword("not-a-valid-hash", "anything"); err == nil {
		t.Fatal("VerifyPassword with malformed hash should error, got nil")
	}
}

// TestVerifyPassword_LegacyBcrypt covers the migrated-user path: accounts
// carried over from the old juanleme/auth.users table have Supabase
// Auth's bcrypt hashes, not our own argon2id ones.
func TestVerifyPassword_LegacyBcrypt(t *testing.T) {
	bcryptHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword: %v", err)
	}
	encoded := string(bcryptHash)

	if !IsBcryptHash(encoded) {
		t.Fatalf("IsBcryptHash(%q) = false, want true", encoded)
	}
	if got, err := VerifyPassword(encoded, "old-password"); err != nil || !got {
		t.Fatalf("VerifyPassword(correct legacy bcrypt) = %v, %v; want true, nil", got, err)
	}
	if got, err := VerifyPassword(encoded, "wrong-password"); err != nil || got {
		t.Fatalf("VerifyPassword(wrong legacy bcrypt) = %v, %v; want false, nil", got, err)
	}
	if IsBcryptHash("not-a-hash") {
		t.Fatal("IsBcryptHash(\"not-a-hash\") = true, want false")
	}
}
