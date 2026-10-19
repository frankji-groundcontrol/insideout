package store

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Whole-change review 4, roadmap expand (rv4-FR-F198, F116, F126, F204).
// ExpandRoadmapNode is the AI-expand write behind POST /roadmap/{nid}/expand/confirm.
// It checked workspace membership only and took no project lock, while every
// other plan writer (CreateRoadmapNode, MoveRoadmapNode, the proposal apply)
// goes through requirePlanWriter and the per-project advisory lock (BUG-020).
// These tests run on real PostgreSQL and use the sibling write as the yardstick.

// rvExpandWorld is one workspace: its owner (workspace admin, project owner), a
// plain member (the viewer-level role), a stranger, one project, and one node
// to expand.
type rvExpandWorld struct {
	owner, member, stranger *User
	project                 *Project
	parent                  *RoadmapNode
}

func rvExpandNewWorld(t *testing.T, st *Store) rvExpandWorld {
	t.Helper()
	ctx := context.Background()
	w := rvExpandWorld{owner: mkUser(t, st), member: mkUser(t, st), stranger: mkUser(t, st)}
	ws, err := st.CreateWorkspace(ctx, w.owner.ID, "Expand review WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, w.member.ID, ws.Code); err != nil {
		t.Fatalf("member joins: %v", err)
	}
	if w.project, err = st.CreateProject(ctx, w.owner.ID, ws.ID, "Expand review", ""); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if w.parent, err = st.CreateRoadmapNode(ctx, w.owner.ID, w.project.ID, nil, "Parent", ""); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	return w
}

// rvExpandChildren returns the nodes directly under the world's parent.
func rvExpandChildren(t *testing.T, st *Store, w rvExpandWorld) []RoadmapNode {
	t.Helper()
	nodes, err := st.ListRoadmap(context.Background(), w.owner.ID, w.project.ID)
	if err != nil {
		t.Fatalf("list roadmap: %v", err)
	}
	var out []RoadmapNode
	for _, n := range nodes {
		if n.ParentID != nil && *n.ParentID == w.parent.ID {
			out = append(out, n)
		}
	}
	return out
}

// A plain member cannot add a node, and so cannot add six through the AI
// expansion either. Who can: the project owner and a workspace admin.
func TestRvExpandNeedsAPlanWriter(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	w := rvExpandNewWorld(t, st)
	kids := []RoadmapPlanNode{{Title: "m1"}, {Title: "m2"}}

	if _, err := st.CreateRoadmapNode(ctx, w.member.ID, w.project.ID, &w.parent.ID, "m0", ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member CreateRoadmapNode (the yardstick): want ErrForbidden, got %v", err)
	}
	if got, err := st.ExpandRoadmapNode(ctx, w.member.ID, w.parent.ID, kids); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member ExpandRoadmapNode: want ErrForbidden, got %d nodes and err %v", len(got), err)
	}
	if _, err := st.ExpandRoadmapNode(ctx, w.member.ID, w.parent.ID, nil); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member ExpandRoadmapNode with nothing to add: want ErrForbidden, got %v", err)
	}
	if n := len(rvExpandChildren(t, st, w)); n != 0 {
		t.Fatalf("the refused expansions left %d children under the parent", n)
	}

	if _, err := st.ExpandRoadmapNode(ctx, w.stranger.ID, w.parent.ID, kids); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stranger ExpandRoadmapNode: want ErrNotFound (existence not disclosed), got %v", err)
	}
	if _, err := st.ExpandRoadmapNode(ctx, w.owner.ID, uuid.New(), kids); !errors.Is(err, ErrNotFound) {
		t.Fatalf("owner ExpandRoadmapNode on an unknown node: want ErrNotFound, got %v", err)
	}

	created, err := st.ExpandRoadmapNode(ctx, w.owner.ID, w.parent.ID, kids)
	if err != nil || len(created) != 2 || created[0].Position != 0 || created[1].Position != 1 {
		t.Fatalf("owner ExpandRoadmapNode: want 2 children at positions 0,1, got %+v, err %v", created, err)
	}
	if err := st.UpdateMemberRole(ctx, w.owner.ID, w.project.WorkspaceID, w.member.ID, "admin"); err != nil {
		t.Fatalf("promote member: %v", err)
	}
	created, err = st.ExpandRoadmapNode(ctx, w.member.ID, w.parent.ID, []RoadmapPlanNode{{Title: "m3"}})
	if err != nil || len(created) != 1 || created[0].Position != 2 {
		t.Fatalf("admin ExpandRoadmapNode: want 1 child at position 2, got %+v, err %v", created, err)
	}
}

// The plan lock is the per-project advisory lock createRoadmapNodeTx,
// MoveRoadmapNode and ReplaceRoadmapTree take. While another writer holds it, an
// expansion has to wait: it reads MAX(position)+1 and a lock-free read would
// give the same answer to two writers. Holding the lock open makes the wait
// observable without depending on a race.
func TestRvExpandWaitsForTheProjectPlanLock(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	w := rvExpandNewWorld(t, st)

	held, release, holder := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		holder <- st.withUserContext(ctx, w.owner.ID, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, w.project.ID.String()); err != nil {
				return err
			}
			close(held)
			<-release
			return nil
		})
	}()
	select {
	case <-held:
	case err := <-holder:
		t.Fatalf("the lock holder did not start: %v", err)
	}

	result := make(chan error, 1)
	go func() {
		_, err := st.ExpandRoadmapNode(ctx, w.owner.ID, w.parent.ID, []RoadmapPlanNode{{Title: "waits"}})
		result <- err
	}()

	var got error
	returned := false
	select {
	case got = <-result:
		returned = true
		t.Errorf("ExpandRoadmapNode returned (%v) while another writer held the project lock, want it to wait", got)
	case <-time.After(400 * time.Millisecond):
	}

	close(release)
	if err := <-holder; err != nil {
		t.Fatalf("the lock holder: %v", err)
	}
	if !returned {
		got = <-result
	}
	if got != nil {
		t.Errorf("ExpandRoadmapNode after the lock was released: %v", got)
	}
}

// Expansions racing each other and racing hand-added nodes under one parent
// must each get their own sibling position: the only index on
// (project_id, parent_id, position) is not unique, so the lock is what keeps the
// order unambiguous.
func TestRvExpandAndCreateNeverShareASiblingPosition(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	w := rvExpandNewWorld(t, st)

	const workers, perWorker = 8, 10
	start := make(chan struct{})
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			if i%2 == 0 { // an AI expansion confirmed in one transaction
				kids := make([]RoadmapPlanNode, perWorker)
				for k := range kids {
					kids[k] = RoadmapPlanNode{Title: fmt.Sprintf("expand-%d-%d", i, k)}
				}
				_, err := st.ExpandRoadmapNode(ctx, w.owner.ID, w.parent.ID, kids)
				errs <- err
				return
			}
			for k := 0; k < perWorker; k++ { // and the same number added by hand
				if _, err := st.CreateRoadmapNode(ctx, w.owner.ID, w.project.ID, &w.parent.ID, fmt.Sprintf("create-%d-%d", i, k), ""); err != nil {
					errs <- err
					return
				}
			}
			errs <- nil
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("a writer failed: %v", err)
		}
	}

	kids := rvExpandChildren(t, st, w)
	if len(kids) != workers*perWorker {
		t.Fatalf("children under the parent = %d, want %d", len(kids), workers*perWorker)
	}
	seen, shared, example := map[int]string{}, 0, ""
	for _, n := range kids {
		if first, dup := seen[n.Position]; dup {
			if shared++; example == "" {
				example = fmt.Sprintf("position %d is shared by %q and %q", n.Position, first, n.Title)
			}
		}
		seen[n.Position] = n.Title
	}
	if shared > 0 {
		t.Errorf("%d of %d siblings share a position with another sibling, e.g. %s", shared, len(kids), example)
	}
}
