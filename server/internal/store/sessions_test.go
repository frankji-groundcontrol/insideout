package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

// F1: rotating an already-rotated (or concurrently-rotated) refresh token must
// NOT mint a second live session. Before the guard, RotateSession's revoking
// UPDATE had no `AND revoked_at IS NULL` and discarded the row count, so a
// replayed token revoked-then-inserted again, leaving two valid sessions from
// one token. Replaying the old session id now affects zero rows → ErrConflict,
// and only the one legitimate successor is active.
func TestSessions_RotateIsReuseSafe(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u := mkUser(t, st)
	exp := time.Now().Add(time.Hour)

	// token_hash is under a TABLE-GLOBAL unique constraint and rotation revokes
	// rows rather than deleting them, so fixed literals would collide with a
	// previous run's still-present revoked row on any re-run. Suffix with the
	// per-run user id (a fresh UUID) so each run's hashes are unique.
	sfx := u.ID.String()
	h0, h1, h2 := "hash-0-"+sfx, "hash-1-"+sfx, "hash-2-"+sfx

	s0, err := st.CreateSession(ctx, u.ID, h0, exp)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// First rotation: s0 → s1. Legitimate, succeeds.
	s1, err := st.RotateSession(ctx, s0.ID, u.ID, h1, exp)
	if err != nil {
		t.Fatalf("first rotate: %v", err)
	}
	if s1.ID == s0.ID {
		t.Fatalf("rotate returned the old session id")
	}

	// Replay: present the already-rotated s0 again (a concurrent refresh or a
	// stolen token). Must refuse with ErrConflict and mint nothing.
	if _, err := st.RotateSession(ctx, s0.ID, u.ID, h2, exp); !errors.Is(err, ErrConflict) {
		t.Fatalf("replay rotate: want ErrConflict, got %v", err)
	}

	// Exactly one successor is live: s1. The replay's h2 was never minted, and
	// the original h0 is revoked — both fail closed.
	if _, err := st.GetActiveSessionByHash(ctx, h1); err != nil {
		t.Fatalf("successor session should be active: %v", err)
	}
	if _, err := st.GetActiveSessionByHash(ctx, h2); !errors.Is(err, ErrNotFound) {
		t.Fatalf("replay must not have minted a session: want ErrNotFound, got %v", err)
	}
	if _, err := st.GetActiveSessionByHash(ctx, h0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("original session must be revoked: want ErrNotFound, got %v", err)
	}
}
