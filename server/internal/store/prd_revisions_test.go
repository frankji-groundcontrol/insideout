package store

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
)

// mkRevisionPrd creates a workspace, an idea and its converted PRD, and
// returns the PRD with its author (the only user who may snapshot it).
func mkRevisionPrd(t *testing.T, st *Store) (*User, *Prd) {
	t.Helper()
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
	return author, prd
}

// Concurrent snapshots of one PRD used to read the same current_revision, all
// compute +1, and leave the losers to trip UNIQUE(prd_id, revision) — a
// spurious 409 although nothing was wrong (F14 only made it a clean 409
// instead of a 500). CreateRevision now queues them behind a per-PRD advisory
// lock, so every call succeeds and takes its own consecutive number.
func TestPrdRevisions_ConcurrentSnapshotsSerialize(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	author, prd := mkRevisionPrd(t, st)

	const n = 8
	var wg sync.WaitGroup
	start := make(chan struct{}) // release all at once to force the read→insert window to overlap
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

	var failed []error
	for _, e := range errs {
		if e != nil {
			failed = append(failed, e)
		}
	}
	if len(failed) > 0 {
		t.Fatalf("%d of %d concurrent snapshots failed, want none (they must queue, not conflict); first: %v", len(failed), n, failed[0])
	}

	// A fresh PRD sits at revision 1 (the column default, no row), so the n
	// snapshots take 2..n+1 with no gap and no duplicate. ListRevisions
	// returns newest first.
	revs, err := st.ListRevisions(ctx, author.ID, prd.ID)
	if err != nil {
		t.Fatalf("list revisions: %v", err)
	}
	if len(revs) != n {
		t.Fatalf("revision rows = %d, want %d", len(revs), n)
	}
	for i, r := range revs {
		if want := n + 1 - i; r.Revision != want {
			t.Fatalf("revs[%d].Revision = %d, want %d (distinct and consecutive)", i, r.Revision, want)
		}
	}
	got, err := st.GetPrdForMember(ctx, prd.ID, author.ID)
	if err != nil {
		t.Fatalf("get prd: %v", err)
	}
	if got.CurrentRevision != n+1 {
		t.Fatalf("current_revision = %d, want %d", got.CurrentRevision, n+1)
	}
}

// The advisory lock only orders writers that take it; UNIQUE(prd_id,
// revision) stays the last line of defence. Plant a row at the next number
// without bumping current_revision — what a half-done race from a writer that
// skips the lock leaves behind — and check the collision still surfaces as a
// typed ErrConflict (409), not a raw 23505 (500).
func TestPrdRevisions_UniqueViolationMapsToConflict(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	author, prd := mkRevisionPrd(t, st)

	if err := st.withUserContext(ctx, author.ID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO insideout.prd_revisions (prd_id, revision, sections, created_by)
			VALUES ($1, $2, '{}'::jsonb, $3)`, prd.ID, prd.CurrentRevision+1, author.ID)
		return err
	}); err != nil {
		t.Fatalf("plant colliding revision: %v", err)
	}

	if _, err := st.CreateRevision(ctx, author.ID, prd.ID, nil); !errors.Is(err, ErrConflict) {
		t.Fatalf("colliding snapshot: want ErrConflict, got %v", err)
	}
}
