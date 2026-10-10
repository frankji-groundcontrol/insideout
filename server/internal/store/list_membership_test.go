package store

import (
	"context"
	"errors"
	"testing"
)

// ARCH-08: a list endpoint's membership rule belongs to the store, re-checked
// in the same transaction as the read (as ListMembers does), not to whichever
// handler remembered a GetMembership precheck. A non-member — a stranger, or
// someone who has since left — gets ErrForbidden and no rows, never an empty
// list that reads as "this workspace has nothing in it".

// wantListForbidden reports (without stopping the test, so every non-member
// case shows up in one run) unless err is ErrForbidden and nothing came back.
func wantListForbidden(t *testing.T, who string, err error, rows int) {
	t.Helper()
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("%s: want ErrForbidden, got err=%v with %d rows", who, err, rows)
	}
	if rows != 0 {
		t.Errorf("%s: returned %d rows alongside the error, want none", who, rows)
	}
}

func TestListIdeas_MemberOnly(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	member := mkUser(t, st)
	stranger := mkUser(t, st)

	ws, err := st.CreateWorkspace(ctx, admin.ID, "List Ideas WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, member.ID, ws.Code); err != nil {
		t.Fatalf("join workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, member.ID, ws.ID, "Mine", "kept")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}

	got, err := st.ListIdeas(ctx, member.ID, ws.ID)
	if err != nil {
		t.Fatalf("member ListIdeas: %v", err)
	}
	if len(got) != 1 || got[0].ID != idea.ID {
		t.Fatalf("member's own list = %v, want exactly their idea", got)
	}

	got, err = st.ListIdeas(ctx, stranger.ID, ws.ID)
	wantListForbidden(t, "stranger ListIdeas", err, len(got))

	// Leaving revokes the author's own inbox too.
	if err := st.RemoveMember(ctx, member.ID, ws.ID, member.ID); err != nil {
		t.Fatalf("member leaves: %v", err)
	}
	got, err = st.ListIdeas(ctx, member.ID, ws.ID)
	wantListForbidden(t, "former member ListIdeas", err, len(got))
}

func TestListProjectsForWorkspace_MemberOnly(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	member := mkUser(t, st)
	stranger := mkUser(t, st)

	ws, err := st.CreateWorkspace(ctx, admin.ID, "List Projects WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, member.ID, ws.Code); err != nil {
		t.Fatalf("join workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Board project", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	for _, viewer := range []*User{admin, member} {
		got, err := st.ListProjectsForWorkspace(ctx, viewer.ID, ws.ID)
		if err != nil {
			t.Fatalf("member ListProjectsForWorkspace: %v", err)
		}
		if len(got) != 1 || got[0].ID != proj.ID {
			t.Fatalf("member's board = %v, want exactly the project", got)
		}
	}

	got, err := st.ListProjectsForWorkspace(ctx, stranger.ID, ws.ID)
	wantListForbidden(t, "stranger ListProjectsForWorkspace", err, len(got))

	if err := st.RemoveMember(ctx, member.ID, ws.ID, member.ID); err != nil {
		t.Fatalf("member leaves: %v", err)
	}
	got, err = st.ListProjectsForWorkspace(ctx, member.ID, ws.ID)
	wantListForbidden(t, "former member ListProjectsForWorkspace", err, len(got))
}
