package store

import (
	"context"
	"testing"
	"time"
)

// TestReapStaleAIRuns_MarksOnlyStaleUnterminated verifies the reaper (§5.3
// of docs/plans/2026-07-21-prd-agent-harness/plan.md) against real RLS:
// the reaper runs with no authenticated actor, so this also exercises the
// 20260721160000_ai_runs_reaper_system_context.sql policy fix — without
// it, the UPDATE matches zero rows under FORCE ROW LEVEL SECURITY.
func TestReapStaleAIRuns_MarksOnlyStaleUnterminated(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	user := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, user.ID, "Reaper Test WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	seed := func(status string, age time.Duration) string {
		var id string
		err := st.Pool.QueryRow(ctx, `
			INSERT INTO insideout.ai_runs (workspace_id, user_id, prompt, status, updated_at)
			VALUES ($1, $2, 'reaper test', $3, now() - $4::interval)
			RETURNING id`,
			ws.ID, user.ID, status, age.String(),
		).Scan(&id)
		if err != nil {
			t.Fatalf("seed ai_run status=%s: %v", status, err)
		}
		return id
	}

	stale := seed("running", 20*time.Minute)
	fresh := seed("running", 2*time.Minute)      // within a heartbeat window — must not be reaped
	pending := seed("pending", 15*time.Minute)   // pending counts too
	already := seed("succeeded", 20*time.Minute) // terminal already — must not be touched

	n, err := st.ReapStaleAIRuns(ctx, 10*time.Minute)
	if err != nil {
		t.Fatalf("reap: %v", err)
	}
	if n < 2 {
		t.Fatalf("reaped count = %d, want at least 2 (stale + pending)", n)
	}

	assertStatus := func(id, want string) {
		var got string
		if err := st.Pool.QueryRow(ctx, `SELECT status FROM insideout.ai_runs WHERE id = $1`, id).Scan(&got); err != nil {
			t.Fatalf("read back %s: %v", id, err)
		}
		if got != want {
			t.Fatalf("run %s status = %q, want %q", id, got, want)
		}
	}
	assertStatus(stale, "failed")
	assertStatus(pending, "failed")
	assertStatus(fresh, "running")     // heartbeat within window — reaper must not kill a healthy long turn
	assertStatus(already, "succeeded") // reaper never touches terminal rows
}

// TestTouchAIRun_UpdatesHeartbeat verifies the heartbeat write runLoop
// makes before every provider call actually advances updated_at, which is
// what lets the reaper tell a long-but-healthy turn from a crashed one.
func TestTouchAIRun_UpdatesHeartbeat(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	user := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, user.ID, "Heartbeat Test WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	run, err := st.CreateAIRun(ctx, ws.ID, nil, user.ID, "heartbeat test")
	if err != nil {
		t.Fatalf("create ai run: %v", err)
	}

	var before time.Time
	if err := st.Pool.QueryRow(ctx, `SELECT updated_at FROM insideout.ai_runs WHERE id = $1`, run.ID).Scan(&before); err != nil {
		t.Fatalf("read initial updated_at: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	if err := st.TouchAIRun(ctx, user.ID, run.ID); err != nil {
		t.Fatalf("touch ai run: %v", err)
	}

	var after time.Time
	if err := st.Pool.QueryRow(ctx, `SELECT updated_at FROM insideout.ai_runs WHERE id = $1`, run.ID).Scan(&after); err != nil {
		t.Fatalf("read updated_at after touch: %v", err)
	}
	if !after.After(before) {
		t.Fatalf("updated_at did not advance: before=%v after=%v", before, after)
	}
}

// TestUpdateSections_CASConflict verifies the fix for the human-vs-coach
// section-edit race (plan §5.4): a stale write (wrong expectedUpdatedAt)
// is rejected with ErrConflict, a fresh one succeeds, and a manual save
// (expectedUpdatedAt=nil, what the PATCH endpoint uses) always wins —
// "the human always wins" is a property of this test, not a comment.
func TestUpdateSections_CASConflict(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	author := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, author.ID, "CAS Test WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, author.ID, ws.ID, "an idea", "content")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	prd, _, err := st.ConvertIdea(ctx, author.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}
	staleRev := prd.UpdatedAt

	// Simulate a human's manual save landing first (no CAS check).
	if _, err := st.UpdateSections(ctx, author.ID, prd.ID, nil, map[string]string{"background": "human wrote this"}, nil); err != nil {
		t.Fatalf("manual save: %v", err)
	}

	// The coach's tool call, still holding the pre-human-edit rev, must
	// be rejected — not silently overwrite the human's edit.
	if _, err := st.UpdateSections(ctx, author.ID, prd.ID, nil, map[string]string{"background": "coach tries to overwrite"}, &staleRev); err != ErrConflict {
		t.Fatalf("stale CAS write: want ErrConflict, got %v", err)
	}

	current, err := st.GetPrdForMember(ctx, prd.ID, author.ID)
	if err != nil {
		t.Fatalf("get prd: %v", err)
	}
	if current.Sections["background"] != "human wrote this" {
		t.Fatalf("background = %q, the rejected coach write must not have applied", current.Sections["background"])
	}

	// A coach write with the CURRENT rev succeeds normally.
	if _, err := st.UpdateSections(ctx, author.ID, prd.ID, nil, map[string]string{"background": "coach re-read then wrote"}, &current.UpdatedAt); err != nil {
		t.Fatalf("fresh CAS write: %v", err)
	}
}

// TestUpdateSections_OptionalTitle verifies the F4 fix: a section-only edit
// (title=nil) must leave the stored title untouched, and only a request that
// deliberately carries a title changes it. Before this, every section save
// resent the client's (possibly stale) title and so could clobber a concurrent
// rename — the PRD title lost-update.
func TestUpdateSections_OptionalTitle(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	author := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, author.ID, "OptTitle WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, author.ID, ws.ID, "original title", "content")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	prd, _, err := st.ConvertIdea(ctx, author.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}
	origTitle := prd.Title

	// A section-only save (nil title) writes the section and leaves the title.
	got, err := st.UpdateSections(ctx, author.ID, prd.ID, nil, map[string]string{"goals": "ship it"}, nil)
	if err != nil {
		t.Fatalf("section-only save: %v", err)
	}
	if got.Title != origTitle {
		t.Fatalf("nil title must not change it: got %q, want %q", got.Title, origTitle)
	}
	if got.Sections["goals"] != "ship it" {
		t.Fatalf("section not written: %q", got.Sections["goals"])
	}

	// An explicit title renames the PRD...
	newTitle := "renamed by hand"
	got, err = st.UpdateSections(ctx, author.ID, prd.ID, &newTitle, nil, nil)
	if err != nil {
		t.Fatalf("title save: %v", err)
	}
	if got.Title != newTitle {
		t.Fatalf("explicit title not applied: got %q, want %q", got.Title, newTitle)
	}
	// ...without clobbering the section written earlier.
	if got.Sections["goals"] != "ship it" {
		t.Fatalf("title-only write clobbered a section: %q", got.Sections["goals"])
	}
}
