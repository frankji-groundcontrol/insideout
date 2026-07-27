package store

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
)

// R1: ConvertIdea used to read the idea (no lock), check status=='converted',
// then insert a PRD + conversation and flip the status. Two concurrent
// converters both read status='pending', both passed the check, and both
// committed their own PRD — orphaning one, since the idea's prd_id can only
// point at one. The fix claims the conversion with an atomic conditional
// UPDATE (status <> 'converted') BEFORE inserting anything, so concurrent
// UPDATEs on the row serialize and exactly one converter wins.
func TestConvertIdea_ConcurrentConvert(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	author := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, author.ID, "Convert WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, author.ID, ws.ID, "Idea", "content")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}

	const n = 8
	var wg sync.WaitGroup
	start := make(chan struct{}) // release all at once to force the race
	prds := make([]*Prd, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			prds[i], _, errs[i] = st.ConvertIdea(ctx, author.ID, idea.ID)
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
			t.Fatalf("opaque error leaked (not nil, not ErrConflict): %v", e)
		}
	}
	// Unlike the revision race (F14), this claims a single resource — the
	// conditional UPDATE serializes on the row, so exactly one converter wins
	// and every other call is a clean, retryable ErrConflict.
	if winners != 1 {
		t.Fatalf("winners = %d, want exactly 1 (%d conflicts)", winners, conflicts)
	}

	// The actual bug was an orphaned duplicate PRD row. Count what committed:
	// exactly one PRD may reference the idea, and the winner's returned PRD
	// must be it. Count through the actor-context path, NOT a raw pool query —
	// prds is FORCE RLS, so an actorless raw SELECT sees zero rows even when a
	// winner's PRD exists.
	var prdCount int
	if err := st.withUserContext(ctx, author.ID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT COUNT(*) FROM insideout.prds WHERE idea_id = $1`, idea.ID).Scan(&prdCount)
	}); err != nil {
		t.Fatalf("count prds: %v", err)
	}
	if prdCount != 1 {
		t.Fatalf("prd rows for idea = %d, want exactly 1 (no orphan from a lost race)", prdCount)
	}

	// The idea ends up converted and pointing at the one winning PRD.
	got, err := st.GetIdeaForMember(ctx, idea.ID, author.ID)
	if err != nil {
		t.Fatalf("get idea: %v", err)
	}
	if got.Status != "converted" {
		t.Fatalf("idea status = %q, want converted", got.Status)
	}
	if got.PrdID == nil || *got.PrdID != prds[indexOfWinner(prds)].ID {
		t.Fatalf("idea.prd_id does not point at the winning PRD")
	}
}

func indexOfWinner(prds []*Prd) int {
	for i, p := range prds {
		if p != nil {
			return i
		}
	}
	return -1
}
