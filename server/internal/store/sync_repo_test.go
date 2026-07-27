package store

import (
	"context"
	"testing"
)

// TestSyncRepoCommits_Atomic verifies the F6 fix: a GitHub sync writes its
// timeline entries and advances the cursor in ONE transaction. The test
// asserts the observable contract — batch and cursor land together, an empty
// batch leaves the cursor untouched, and an outsider writes nothing.
func TestSyncRepoCommits_Atomic(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	owner := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, owner.ID, "Sync WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, owner.ID, ws.ID, "Sync Proj", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Read through the actor-context store methods: project_updates and
	// projects carry FORCE ROW LEVEL SECURITY, so a bare pool query (no
	// app.user_id) sees nothing. The owner is a member, so these return truth.
	countUpdates := func() int {
		ups, err := st.ListProjectUpdates(ctx, owner.ID, proj.ID, 0, nil)
		if err != nil {
			t.Fatalf("list updates: %v", err)
		}
		return len(ups)
	}
	cursor := func() string {
		_, sha, err := st.ProjectRepoSync(ctx, owner.ID, proj.ID)
		if err != nil {
			t.Fatalf("read cursor: %v", err)
		}
		return sha
	}

	// A sync of two commits inserts both (oldest-first order preserved) and
	// advances the cursor in the same breath.
	added, err := st.SyncRepoCommits(ctx, owner.ID, proj.ID, []string{"first", "second"}, "sha-2")
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if added != 2 {
		t.Fatalf("added = %d, want 2", added)
	}
	if n := countUpdates(); n != 2 {
		t.Fatalf("updates = %d, want 2", n)
	}
	if got := cursor(); got != "sha-2" {
		t.Fatalf("cursor = %q, want %q", got, "sha-2")
	}

	// Nothing new: the cursor must not move.
	added, err = st.SyncRepoCommits(ctx, owner.ID, proj.ID, nil, "sha-should-not-stick")
	if err != nil {
		t.Fatalf("empty sync: %v", err)
	}
	if added != 0 {
		t.Fatalf("empty added = %d, want 0", added)
	}
	if got := cursor(); got != "sha-2" {
		t.Fatalf("cursor moved on empty sync: %q", got)
	}

	// A later sync appends and advances again.
	if _, err := st.SyncRepoCommits(ctx, owner.ID, proj.ID, []string{"third"}, "sha-3"); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if n := countUpdates(); n != 3 {
		t.Fatalf("updates after second sync = %d, want 3", n)
	}
	if got := cursor(); got != "sha-3" {
		t.Fatalf("cursor = %q, want %q", got, "sha-3")
	}

	// An outsider (not owner, not workspace admin) must write nothing.
	outsider := mkUser(t, st)
	if _, err := st.SyncRepoCommits(ctx, outsider.ID, proj.ID, []string{"intruder"}, "sha-bad"); err == nil {
		t.Fatalf("outsider sync succeeded, want error")
	}
	if n := countUpdates(); n != 3 {
		t.Fatalf("outsider wrote rows: updates = %d, want 3", n)
	}
	if got := cursor(); got != "sha-3" {
		t.Fatalf("outsider moved cursor: %q", got)
	}
}
