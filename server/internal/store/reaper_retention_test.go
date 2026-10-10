package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// existsRow reports whether query (an EXISTS SELECT) holds on the bare
// pool — the system context the reaper itself runs under.
func existsRow(t *testing.T, st *Store, query string, args ...any) bool {
	t.Helper()
	var exists bool
	if err := st.Pool.QueryRow(context.Background(), query, args...).Scan(&exists); err != nil {
		t.Fatalf("exists check: %v", err)
	}
	return exists
}

// REL-03: the reaper must not only flip stale ai_runs to failed — rows past
// a retention window have to be purged (ai_runs with their raw prompt/
// response text, cascading ai_run_events telemetry, and long-expired
// sessions, auth_tokens and invitations), or the schema grows without
// bound. Seeds rows 45 days old plus fresh counterparts, runs the same
// entry point main.go's ticker calls, and checks only the old ones went.
func TestReapStaleAIRuns_PurgesRowsPastRetention(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Retention WS "+uuid.NewString()[:8], "desc")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	oldRun, err := st.CreateAIRun(ctx, ws.ID, nil, admin.ID, "old prompt")
	if err != nil {
		t.Fatalf("create old ai_run: %v", err)
	}
	if _, err := st.Pool.Exec(ctx, `
		UPDATE insideout.ai_runs
		SET created_at = now() - interval '45 days', updated_at = now() - interval '45 days'
		WHERE id = $1`, oldRun.ID,
	); err != nil {
		t.Fatalf("backdate old ai_run: %v", err)
	}
	// Telemetry row on the old run: it must cascade away with the run.
	if err := st.InsertAIRunEvent(ctx, oldRun.ID, "call", "test-model", nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("insert ai_run_event: %v", err)
	}
	freshRun, err := st.CreateAIRun(ctx, ws.ID, nil, admin.ID, "fresh prompt")
	if err != nil {
		t.Fatalf("create fresh ai_run: %v", err)
	}

	// Sessions: one that expired 45 days ago, one live for another 30 days.
	if _, err := st.CreateSession(ctx, admin.ID, "retention-old-"+uuid.NewString(), time.Now().Add(-45*24*time.Hour)); err != nil {
		t.Fatalf("create expired session: %v", err)
	}
	if _, err := st.CreateSession(ctx, admin.ID, "retention-live-"+uuid.NewString(), time.Now().Add(30*24*time.Hour)); err != nil {
		t.Fatalf("create live session: %v", err)
	}

	// Auth tokens: different kinds so the one-live-per-kind discipline of
	// CreateAuthToken does not drop the old row when the live one is minted.
	if err := st.CreateAuthToken(ctx, admin.ID, AuthTokenMagicLink, "retention-old-"+uuid.NewString(), time.Now().Add(-45*24*time.Hour)); err != nil {
		t.Fatalf("create expired auth token: %v", err)
	}
	liveTokenHash := "retention-live-" + uuid.NewString()
	if err := st.CreateAuthToken(ctx, admin.ID, AuthTokenPasswordReset, liveTokenHash, time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("create live auth token: %v", err)
	}

	// Invitations: distinct addresses (CreateInvitation supersedes the
	// prior unused invite for the same email).
	if _, err := st.CreateInvitation(ctx, admin.ID, "retention-old-"+uuid.NewString()+"@test.local", "old-"+uuid.NewString(), time.Now().Add(-45*24*time.Hour)); err != nil {
		t.Fatalf("create expired invitation: %v", err)
	}
	if _, err := st.CreateInvitation(ctx, admin.ID, "retention-live-"+uuid.NewString()+"@test.local", "live-"+uuid.NewString(), time.Now().Add(7*24*time.Hour)); err != nil {
		t.Fatalf("create live invitation: %v", err)
	}

	if _, err := st.ReapStaleAIRuns(ctx, 10*time.Minute); err != nil {
		t.Fatalf("reap: %v", err)
	}

	if existsRow(t, st, `SELECT EXISTS (SELECT 1 FROM insideout.ai_runs WHERE id = $1)`, oldRun.ID) {
		t.Error("ai_run 45 days old still present: the reaper only flips status and never deletes rows")
	}
	if existsRow(t, st, `SELECT EXISTS (SELECT 1 FROM insideout.ai_run_events WHERE run_id = $1)`, oldRun.ID) {
		t.Error("ai_run_events row for the purged run still present: the cascade never fired")
	}
	if !existsRow(t, st, `SELECT EXISTS (SELECT 1 FROM insideout.ai_runs WHERE id = $1)`, freshRun.ID) {
		t.Error("fresh ai_run was purged")
	}
	if existsRow(t, st, `SELECT EXISTS (SELECT 1 FROM insideout.sessions WHERE token_hash LIKE 'retention-old-%')`) {
		t.Error("session expired 45 days ago still present: no retention purge for sessions")
	}
	if !existsRow(t, st, `SELECT EXISTS (SELECT 1 FROM insideout.sessions WHERE token_hash LIKE 'retention-live-%')`) {
		t.Error("live session was purged")
	}
	if existsRow(t, st, `SELECT EXISTS (SELECT 1 FROM insideout.auth_tokens WHERE token_hash LIKE 'retention-old-%')`) {
		t.Error("auth_token expired 45 days ago still present: no retention purge for auth_tokens")
	}
	if !existsRow(t, st, `SELECT EXISTS (SELECT 1 FROM insideout.auth_tokens WHERE token_hash = $1)`, liveTokenHash) {
		t.Error("live auth_token was purged")
	}
	if existsRow(t, st, `SELECT EXISTS (SELECT 1 FROM insideout.invitations WHERE email LIKE 'retention-old-%')`) {
		t.Error("invitation expired 45 days ago still present: no retention purge for invitations")
	}
	if !existsRow(t, st, `SELECT EXISTS (SELECT 1 FROM insideout.invitations WHERE email LIKE 'retention-live-%')`) {
		t.Error("live invitation was purged")
	}
}
