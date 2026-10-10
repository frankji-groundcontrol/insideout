package store

import (
	"context"
	"sync"
	"testing"
)

// CreatePrdCommit and CreateRevision both number the next revision from
// prds.current_revision, but each used to take a different lock first (the PRD
// row versus an advisory lock). A commit racing a snapshot could then collide
// on UNIQUE(prd_id, revision) or, with the opposite lock order, deadlock on
// the row lock. Both now queue behind the per-PRD advisory lock.
func TestPrdCommitAndSnapshotRaceDoNotFail(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	author, prd := mkRevisionPrd(t, st)

	const each = 6
	var wg sync.WaitGroup
	start := make(chan struct{}) // release all at once so the read→insert windows overlap
	errs := make([]error, 2*each)
	for i := 0; i < each; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = st.CreateRevision(ctx, author.ID, prd.ID, nil)
		}(i)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[each+i] = st.CreatePrdCommit(ctx, author.ID, prd.ID, "v", "team", "summary", nil, "")
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("racing writer %d of %d failed, want all to queue: %v", i, len(errs), err)
		}
	}
	revs, err := st.ListRevisions(ctx, author.ID, prd.ID)
	if err != nil {
		t.Fatalf("list revisions: %v", err)
	}
	if len(revs) != 2*each {
		t.Fatalf("revisions = %d, want %d (each writer takes its own number)", len(revs), 2*each)
	}
}
