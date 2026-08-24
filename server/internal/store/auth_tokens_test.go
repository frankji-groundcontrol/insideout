package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Real-Postgres integration tests for the emailed one-time auth tokens
// (magic link + password reset). Skipped without DATABASE_URL, same gate
// and rationale as authz_test.go. Hashes are uuid-derived per run: rows
// persist after the run (the mkUser convention), and token_hash is
// globally unique, so fixed strings would collide on a second run.

func mustCreateAuthToken(t *testing.T, st *Store, userID uuid.UUID, kind, tokenHash string, ttl time.Duration) {
	t.Helper()
	if err := st.CreateAuthToken(context.Background(), userID, kind, tokenHash, time.Now().Add(ttl)); err != nil {
		t.Fatalf("create auth token: %v", err)
	}
}

func freshHash(t *testing.T) string {
	t.Helper()
	return "hash-" + uuid.NewString()
}

func TestAuthTokenConsumeRoundTrip(t *testing.T) {
	st := newTestStore(t)
	u := mkUser(t, st)

	h := freshHash(t)
	mustCreateAuthToken(t, st, u.ID, AuthTokenMagicLink, h, 15*time.Minute)

	got, err := st.ConsumeAuthToken(context.Background(), AuthTokenMagicLink, h)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if got != u.ID {
		t.Fatalf("consumed userID = %v, want %v", got, u.ID)
	}
}

func TestAuthTokenSingleUse(t *testing.T) {
	st := newTestStore(t)
	u := mkUser(t, st)

	h := freshHash(t)
	mustCreateAuthToken(t, st, u.ID, AuthTokenPasswordReset, h, time.Hour)
	if _, err := st.ConsumeAuthToken(context.Background(), AuthTokenPasswordReset, h); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if _, err := st.ConsumeAuthToken(context.Background(), AuthTokenPasswordReset, h); err != ErrNotFound {
		t.Fatalf("second consume err = %v, want ErrNotFound", err)
	}
}

func TestAuthTokenExpired(t *testing.T) {
	st := newTestStore(t)
	u := mkUser(t, st)

	h := freshHash(t)
	mustCreateAuthToken(t, st, u.ID, AuthTokenMagicLink, h, -time.Minute)
	if _, err := st.ConsumeAuthToken(context.Background(), AuthTokenMagicLink, h); err != ErrNotFound {
		t.Fatalf("expired consume err = %v, want ErrNotFound", err)
	}
}

func TestAuthTokenKindIsolation(t *testing.T) {
	st := newTestStore(t)
	u := mkUser(t, st)

	h := freshHash(t)
	mustCreateAuthToken(t, st, u.ID, AuthTokenMagicLink, h, time.Hour)
	if _, err := st.ConsumeAuthToken(context.Background(), AuthTokenPasswordReset, h); err != ErrNotFound {
		t.Fatalf("wrong-kind consume err = %v, want ErrNotFound", err)
	}
	// The token survives the wrong-kind probe and is still consumable.
	if _, err := st.ConsumeAuthToken(context.Background(), AuthTokenMagicLink, h); err != nil {
		t.Fatalf("right-kind consume after wrong-kind probe: %v", err)
	}
}

func TestAuthTokenCreateReplacesPriorUnused(t *testing.T) {
	st := newTestStore(t)
	u := mkUser(t, st)

	oldHash, newHash := freshHash(t), freshHash(t)
	mustCreateAuthToken(t, st, u.ID, AuthTokenMagicLink, oldHash, 15*time.Minute)
	mustCreateAuthToken(t, st, u.ID, AuthTokenMagicLink, newHash, 15*time.Minute)

	if _, err := st.ConsumeAuthToken(context.Background(), AuthTokenMagicLink, oldHash); err != ErrNotFound {
		t.Fatalf("superseded consume err = %v, want ErrNotFound", err)
	}
	if _, err := st.ConsumeAuthToken(context.Background(), AuthTokenMagicLink, newHash); err != nil {
		t.Fatalf("replacement consume: %v", err)
	}
}

func TestAuthTokenUnknownHash(t *testing.T) {
	st := newTestStore(t)
	if _, err := st.ConsumeAuthToken(context.Background(), AuthTokenMagicLink, freshHash(t)); err != ErrNotFound {
		t.Fatalf("unknown consume err = %v, want ErrNotFound", err)
	}
}

func TestRevokeUserSessions(t *testing.T) {
	st := newTestStore(t)
	u := mkUser(t, st)
	other := mkUser(t, st)

	ctx := context.Background()
	mine, otherHash := "sess-"+uuid.NewString(), "sess-"+uuid.NewString()
	if _, err := st.CreateSession(ctx, u.ID, mine, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := st.CreateSession(ctx, other.ID, otherHash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create other session: %v", err)
	}

	if err := st.RevokeUserSessions(ctx, u.ID); err != nil {
		t.Fatalf("revoke user sessions: %v", err)
	}

	if _, err := st.GetActiveSessionByHash(ctx, mine); err != ErrNotFound {
		t.Fatalf("revoked session err = %v, want ErrNotFound", err)
	}
	if _, err := st.GetActiveSessionByHash(ctx, otherHash); err != nil {
		t.Fatalf("other user's session must survive: %v", err)
	}
}

// Login-code tests: a 6-digit code is low-entropy, so verification runs
// under an attempt budget — misses retire the code, and only the exact
// hash consumes it, same single-use discipline as the link tokens.

func TestLoginCodeVerifyRoundTrip(t *testing.T) {
	st := newTestStore(t)
	u := mkUser(t, st)

	h := freshHash(t)
	mustCreateAuthToken(t, st, u.ID, AuthTokenLoginCode, h, 10*time.Minute)

	if err := st.VerifyLoginCode(context.Background(), u.ID, h); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := st.VerifyLoginCode(context.Background(), u.ID, h); err != ErrNotFound {
		t.Fatalf("second verify err = %v, want ErrNotFound", err)
	}
}

func TestLoginCodeWrongHashMisses(t *testing.T) {
	st := newTestStore(t)
	u := mkUser(t, st)

	h := freshHash(t)
	mustCreateAuthToken(t, st, u.ID, AuthTokenLoginCode, h, 10*time.Minute)

	if err := st.VerifyLoginCode(context.Background(), u.ID, freshHash(t)); err != ErrNotFound {
		t.Fatalf("wrong-hash verify err = %v, want ErrNotFound", err)
	}
	// A miss consumes the budget, not the code: the right hash still works.
	if err := st.VerifyLoginCode(context.Background(), u.ID, h); err != nil {
		t.Fatalf("right-hash verify after a miss: %v", err)
	}
}

func TestLoginCodeAttemptsExhaust(t *testing.T) {
	st := newTestStore(t)
	u := mkUser(t, st)

	h := freshHash(t)
	mustCreateAuthToken(t, st, u.ID, AuthTokenLoginCode, h, 10*time.Minute)

	for i := 0; i < LoginCodeMaxAttempts; i++ {
		if err := st.VerifyLoginCode(context.Background(), u.ID, freshHash(t)); err != ErrNotFound {
			t.Fatalf("miss %d err = %v, want ErrNotFound", i+1, err)
		}
	}
	if err := st.VerifyLoginCode(context.Background(), u.ID, h); err != ErrNotFound {
		t.Fatalf("exhausted code err = %v, want ErrNotFound", err)
	}
}

func TestLoginCodeExpired(t *testing.T) {
	st := newTestStore(t)
	u := mkUser(t, st)

	h := freshHash(t)
	mustCreateAuthToken(t, st, u.ID, AuthTokenLoginCode, h, -time.Minute)
	if err := st.VerifyLoginCode(context.Background(), u.ID, h); err != ErrNotFound {
		t.Fatalf("expired verify err = %v, want ErrNotFound", err)
	}
}

func TestLoginCodeNoActiveCode(t *testing.T) {
	st := newTestStore(t)
	u := mkUser(t, st)

	if err := st.VerifyLoginCode(context.Background(), u.ID, freshHash(t)); err != ErrNotFound {
		t.Fatalf("no-code verify err = %v, want ErrNotFound", err)
	}
}
