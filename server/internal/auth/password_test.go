package auth

import (
	"strings"
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

// VerifyDummyPassword only equalizes login timing if the dummy hash is a
// well-formed hash with the production parameters: a malformed one makes
// VerifyPassword fail at once (no argon2 work), and a stale copy of the
// parameters would make the unknown-email path cheaper or dearer than a real
// verify.
func TestDummyHash_IsRealWork(t *testing.T) {
	if ok, err := VerifyPassword(dummyHash(), "some-other-password"); err != nil || ok {
		t.Fatalf("VerifyPassword(dummy, other) = %v, %v; want false, nil (parseable, not matching)", ok, err)
	}

	real, err := HashPassword("anything")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	r, d := strings.Split(real, "$"), strings.Split(dummyHash(), "$")
	if len(d) != len(r) {
		t.Fatalf("dummy hash has %d $-separated parts, a real one %d", len(d), len(r))
	}
	// ["", "argon2id", "v=19", "m=65536,t=1,p=4", salt, hash]: all that sets the cost.
	for i := 1; i <= 3; i++ {
		if d[i] != r[i] {
			t.Fatalf("dummy hash part %d = %q, want the production %q", i, d[i], r[i])
		}
	}
	if len(d[5]) != len(r[5]) {
		t.Fatalf("dummy hash output length = %d, want %d", len(d[5]), len(r[5]))
	}

	VerifyDummyPassword("anything") // smoke: must not panic
}
