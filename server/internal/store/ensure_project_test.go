package store

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// F3: two concurrent first-builds of the same PRD must create exactly ONE
// project and link it — not two orphaned projects under divergent roadmaps.
// Before the per-PRD advisory lock, EnsureProjectForPrd was a
// read(project_id NULL)-modify-insert race: every overlapping caller read NULL
// and inserted its own project. With the lock, the losers block until the
// winner commits, then all read and return the same linked project.
//
// The read-insert window is a fraction of a millisecond, and a cold pool makes
// the callers queue behind connection setup (the first finishes before the
// rest are even connected), which hides the race — the sibling
// TestEnsureConversationForPrd_ConcurrentCallsLeaveOneRow learned this the hard
// way. Warm the pool first, then repeat the burst on a fresh unlinked PRD each
// round until a missing lock cannot hide.
func TestEnsureProjectForPrd_ConcurrentFirstBuild(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Ensure WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	const callers, rounds = 8, 30
	var release []func()
	for i := 0; i < min(callers, int(st.Pool.Config().MaxConns)); i++ {
		conn, err := st.Pool.Acquire(ctx)
		if err != nil {
			t.Fatalf("warm the pool: %v", err)
		}
		release = append(release, conn.Release)
	}
	for _, r := range release {
		r()
	}

	for round := 0; round < rounds; round++ {
		idea, err := st.CreateIdea(ctx, admin.ID, ws.ID, "Idea", "content")
		if err != nil {
			t.Fatalf("round %d create idea: %v", round, err)
		}
		prd, _, err := st.ConvertIdea(ctx, admin.ID, idea.ID)
		if err != nil {
			t.Fatalf("round %d convert idea: %v", round, err)
		}
		before := countProjects(t, st, ws.ID, admin.ID)

		var wg sync.WaitGroup
		start := make(chan struct{}) // release all goroutines at once to force the race
		ids := make([]uuid.UUID, callers)
		errs := make([]error, callers)
		for i := 0; i < callers; i++ {
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
				t.Fatalf("round %d goroutine %d EnsureProjectForPrd: %v", round, i, err)
			}
		}
		// The invariant that proves no orphan: every caller saw the SAME
		// project. If the race had minted multiple projects, the ids would
		// diverge — and the workspace would hold more than one new project.
		want := ids[0]
		if want == uuid.Nil {
			t.Fatalf("round %d: EnsureProjectForPrd returned a nil project id", round)
		}
		for i, id := range ids {
			if id != want {
				t.Fatalf("round %d: orphaned project: goroutine %d got %s, want %s (concurrent first-builds must converge on one project)", round, i, id, want)
			}
		}
		if n := countProjects(t, st, ws.ID, admin.ID); n != before+1 {
			t.Fatalf("round %d: workspace has %d projects after %d concurrent first-builds, want %d", round, n, callers, before+1)
		}

		// Idempotent on a follow-up call too.
		again, err := st.EnsureProjectForPrd(ctx, admin.ID, prd.ID)
		if err != nil {
			t.Fatalf("round %d serial re-ensure: %v", round, err)
		}
		if again.ID != want {
			t.Fatalf("round %d re-ensure diverged: got %s, want %s", round, again.ID, want)
		}
	}
}

// countProjects counts the workspace's projects under userID's RLS.
func countProjects(t *testing.T, st *Store, workspaceID, userID uuid.UUID) int {
	t.Helper()
	var n int
	err := st.withUserContext(context.Background(), userID, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(),
			`SELECT count(*) FROM insideout.projects WHERE workspace_id = $1`,
			workspaceID,
		).Scan(&n)
	})
	if err != nil {
		t.Fatalf("count projects: %v", err)
	}
	return n
}
