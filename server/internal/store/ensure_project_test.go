package store

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// F3: two concurrent first-builds of the same PRD must create exactly ONE
// project and link it — not two orphaned projects under divergent roadmaps.
// Before the per-PRD advisory lock, EnsureProjectForPrd was a
// read(project_id NULL)-modify-insert race: every overlapping caller read NULL
// and inserted its own project. With the lock, the losers block until the
// winner commits, then all read and return the same linked project.
func TestEnsureProjectForPrd_ConcurrentFirstBuild(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Ensure WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, admin.ID, ws.ID, "Idea", "content")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	prd, _, err := st.ConvertIdea(ctx, admin.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}

	const n = 8
	var wg sync.WaitGroup
	start := make(chan struct{}) // release all goroutines at once to force the race
	ids := make([]uuid.UUID, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			p, err := st.EnsureProjectForPrd(ctx, admin.ID, prd.ID)
			if err != nil {
				errs[i] = err
				return
			}
			ids[i] = p.ID
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d EnsureProjectForPrd: %v", i, err)
		}
	}
	// The one invariant that proves no orphan: every caller saw the SAME
	// project. If the race had minted multiple projects, the ids would diverge.
	want := ids[0]
	if want == uuid.Nil {
		t.Fatal("EnsureProjectForPrd returned a nil project id")
	}
	for i, id := range ids {
		if id != want {
			t.Fatalf("orphaned project: goroutine %d got %s, want %s (concurrent first-builds must converge on one project)", i, id, want)
		}
	}

	// Idempotent on a follow-up call too.
	again, err := st.EnsureProjectForPrd(ctx, admin.ID, prd.ID)
	if err != nil {
		t.Fatalf("serial re-ensure: %v", err)
	}
	if again.ID != want {
		t.Fatalf("re-ensure diverged: got %s, want %s", again.ID, want)
	}
}
