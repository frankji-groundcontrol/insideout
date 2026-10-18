package store

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
)

// rv4-DATA-05: the timeline cursor used to be a bare row id that the SQL
// resolved with a subselect on project_updates. Authors may delete their
// updates, and when the cursor row was gone the comparison was NULL: the next
// page came back empty and a client mid-scroll silently lost all older
// history. The cursor now carries the (created_at, id) pair the order is
// built on, so deleting the row it was taken from changes nothing.

// cursorFixture is a project with five updates; updates[0] is the oldest, so
// the timeline reads updates[4], [3], [2], [1], [0].
type cursorFixture struct {
	st        *Store
	owner     uuid.UUID
	workspace uuid.UUID
	project   uuid.UUID
	updates   []*ProjectUpdate
}

func newCursorFixture(t *testing.T) cursorFixture {
	t.Helper()
	st := newTestStore(t)
	ctx := context.Background()
	owner := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, owner.ID, "Cursor WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, owner.ID, ws.ID, "Cursor Proj", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	f := cursorFixture{st: st, owner: owner.ID, workspace: ws.ID, project: proj.ID}
	for i := 0; i < 5; i++ {
		u, err := st.AddProjectUpdate(ctx, owner.ID, proj.ID, "note", fmt.Sprintf("update-%d", i))
		if err != nil {
			t.Fatalf("add update %d: %v", i, err)
		}
		f.updates = append(f.updates, u)
	}
	return f
}

// wantPage fails the test unless got is exactly the wanted updates, in order.
func wantPage(t *testing.T, label string, got []ProjectUpdate, want ...*ProjectUpdate) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: %d rows, want %d", label, len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i].ID {
			t.Fatalf("%s: row %d is %q, want %q", label, i, got[i].Content, want[i].Content)
		}
	}
}

func TestProjectUpdates_CursorSurvivesADeletedRow(t *testing.T) {
	f := newCursorFixture(t)
	ctx := context.Background()
	u := f.updates

	page1, err := f.st.ListProjectUpdatesBefore(ctx, f.owner, f.project, 2, nil)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	wantPage(t, "page 1", page1, u[4], u[3])
	cursor := page1[len(page1)-1].Cursor()

	// The author deletes the row the cursor was taken from, mid-scroll.
	if err := f.st.DeleteProjectUpdate(ctx, f.owner, u[3].ID); err != nil {
		t.Fatalf("delete the cursor row: %v", err)
	}

	// The older history must still be there: the subselect cursor returned
	// an empty page here and the client lost everything below.
	page2, err := f.st.ListProjectUpdatesBefore(ctx, f.owner, f.project, 2, &cursor)
	if err != nil {
		t.Fatalf("page 2 after the cursor row was deleted: %v", err)
	}
	wantPage(t, "page 2 after the cursor row was deleted", page2, u[2], u[1])

	cursor = page2[len(page2)-1].Cursor()
	page3, err := f.st.ListProjectUpdatesBefore(ctx, f.owner, f.project, 2, &cursor)
	if err != nil {
		t.Fatalf("page 3: %v", err)
	}
	wantPage(t, "page 3", page3, u[0])
}

// The id-only API cannot know where a deleted row sat, so it must say so
// instead of answering with an empty page that looks like the end of history.
func TestProjectUpdates_IDCursorOnAnUnknownRowIsNotFound(t *testing.T) {
	f := newCursorFixture(t)
	ctx := context.Background()
	u := f.updates

	// A live id still pages exactly as before.
	page, err := f.st.ListProjectUpdates(ctx, f.owner, f.project, 2, &u[3].ID)
	if err != nil {
		t.Fatalf("page after a live id: %v", err)
	}
	wantPage(t, "page after a live id", page, u[2], u[1])

	if err := f.st.DeleteProjectUpdate(ctx, f.owner, u[3].ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := f.st.ListProjectUpdates(ctx, f.owner, f.project, 2, &u[3].ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("page after a deleted id: err = %v, want ErrNotFound (not an empty page)", err)
	}

	// An id from another project is not a position in this timeline either.
	other, err := f.st.CreateProject(ctx, f.owner, f.workspace, "Other Proj", "")
	if err != nil {
		t.Fatalf("create the other project: %v", err)
	}
	foreign, err := f.st.AddProjectUpdate(ctx, f.owner, other.ID, "note", "elsewhere")
	if err != nil {
		t.Fatalf("add the foreign update: %v", err)
	}
	if _, err := f.st.ListProjectUpdates(ctx, f.owner, f.project, 2, &foreign.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("page after another project's id: err = %v, want ErrNotFound", err)
	}
}
