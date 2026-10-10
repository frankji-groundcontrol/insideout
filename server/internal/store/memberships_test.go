package store

import (
	"context"
	"errors"
	"testing"
)

// ListMembers is member-only. A member sees the roster (join order); a
// non-member — a stranger, or someone who has since left — gets ErrForbidden,
// never an empty list that reads as "this workspace has no members".
func TestListMembers_MemberOnly(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	member := mkUser(t, st)
	stranger := mkUser(t, st)

	ws, err := st.CreateWorkspace(ctx, admin.ID, "Roster WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, member.ID, ws.Code); err != nil {
		t.Fatalf("join workspace: %v", err)
	}

	for _, viewer := range []*User{admin, member} {
		list, err := st.ListMembers(ctx, viewer.ID, ws.ID)
		if err != nil {
			t.Fatalf("member ListMembers: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("roster has %d rows, want 2", len(list))
		}
		if list[0].UserID != admin.ID || list[0].Role != "admin" {
			t.Fatalf("roster[0] = %s/%s, want the admin first (join order)", list[0].UserID, list[0].Role)
		}
		if list[1].UserID != member.ID || list[1].Role != "member" {
			t.Fatalf("roster[1] = %s/%s, want the member second", list[1].UserID, list[1].Role)
		}
	}

	list, err := st.ListMembers(ctx, stranger.ID, ws.ID)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("stranger ListMembers: want ErrForbidden, got err=%v with %d rows", err, len(list))
	}
	if list != nil {
		t.Fatalf("stranger ListMembers returned %d rows alongside the error, want none", len(list))
	}

	// Leaving revokes the roster too.
	if err := st.RemoveMember(ctx, member.ID, ws.ID, member.ID); err != nil {
		t.Fatalf("member leaves: %v", err)
	}
	if _, err := st.ListMembers(ctx, member.ID, ws.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("former member ListMembers: want ErrForbidden, got %v", err)
	}
}
