package store

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// F14: two concurrent snapshots both read the same current_revision and both
// compute MAX+1. Without the 23505→ErrConflict mapping the loser surfaced an
// opaque Postgres error (500 at the API). With it, exactly one call wins and
// every loser returns ErrConflict (409) — never a raw/unknown error — and the
// winner's single row stands.
func TestPrdRevisions_ConcurrentSnapshotConflict(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	author := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, author.ID, "Revision WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, author.ID, ws.ID, "Idea", "content")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	prd, _, err := st.ConvertIdea(ctx, author.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}

	// Burst n concurrent snapshots at the same PRD. How many genuinely
	// overlap in the read(current_revision)→insert(MAX+1) window is
	// timing-dependent: overlapping callers collide on the (prd_id, revision)
	// unique constraint and the losers must surface ErrConflict; callers that
	// happen to serialize each legitimately claim the next number. The
	// invariant F14 guards is therefore NOT "exactly one winner" but "no call
	// ever returns an opaque error" — every outcome is either a clean success
	// or a typed ErrConflict (409), never a raw 23505 bubbling up as a 500.
	const n = 8
	var wg sync.WaitGroup
	start := make(chan struct{}) // release all at once to force the MAX+1 race
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = st.CreateRevision(ctx, author.ID, prd.ID, nil)
		}(i)
	}
	close(start)
	wg.Wait()

	var winners, conflicts int
	for _, e := range errs {
		switch {
		case e == nil:
			winners++
		case errors.Is(e, ErrConflict):
			conflicts++
		default:
			t.Fatalf("opaque error leaked through (not nil, not ErrConflict): %v", e)
		}
	}
	if winners < 1 {
		t.Fatal("winners = 0, want at least one snapshot to succeed")
	}
	t.Logf("overlap outcome: %d winners, %d conflicts", winners, conflicts)

	// Every winner wrote exactly one row, and the losers rolled back — so the
	// revision count equals the winner count, with no gaps or duplicates.
	revs, err := st.ListRevisions(ctx, author.ID, prd.ID)
	if err != nil {
		t.Fatalf("list revisions: %v", err)
	}
	if len(revs) != winners {
		t.Fatalf("revision rows = %d, want one per winner (%d)", len(revs), winners)
	}
	seen := make(map[int]bool, len(revs))
	for _, r := range revs {
		if seen[r.Revision] {
			t.Fatalf("duplicate revision %d persisted", r.Revision)
		}
		seen[r.Revision] = true
	}
}
