package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

// USER-05 / FEAT-04: ideas are private to their author by default
// (PRODUCT.md: "Capture remains one-step and private to its author by
// default"; the README sells each member a personal inbox). Enforced at
// both layers: the queries filter to the author, and the ideas_select RLS
// policy is author-or-admin — matching the admin's existing drop
// authority.
func TestIdeaPrivacy_AuthorScoped(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	author := mkUser(t, st)
	other := mkUser(t, st)

	ws, err := st.CreateWorkspace(ctx, admin.ID, "Idea Privacy WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	for _, u := range []*User{author, other} {
		if _, err := st.JoinWorkspace(ctx, u.ID, ws.Code); err != nil {
			t.Fatalf("join workspace: %v", err)
		}
	}
	idea, err := st.CreateIdea(ctx, author.ID, ws.ID, "Private thought", "half-formed")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}

	// A fellow member's board does not list the author's ideas.
	got, err := st.ListIdeas(ctx, other.ID, ws.ID)
	if err != nil {
		t.Fatalf("other member ListIdeas: %v", err)
	}
	for _, i := range got {
		if i.ID == idea.ID {
			t.Fatal("fellow member listed the author's private idea")
		}
	}
	// The author still sees their own.
	got, err = st.ListIdeas(ctx, author.ID, ws.ID)
	if err != nil {
		t.Fatalf("author ListIdeas: %v", err)
	}
	if len(got) != 1 || got[0].ID != idea.ID {
		t.Fatalf("author's own list = %v, want exactly their idea", got)
	}

	// Direct fetch by ID hides it from a fellow member (existence not
	// disclosed) but not from the author or the workspace admin.
	if _, err := st.GetIdeaForMember(ctx, idea.ID, other.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("fellow member GetIdeaForMember: want ErrNotFound, got %v", err)
	}
	if _, err := st.GetIdeaForMember(ctx, idea.ID, author.ID); err != nil {
		t.Fatalf("author GetIdeaForMember: %v", err)
	}
	if _, err := st.GetIdeaForMember(ctx, idea.ID, admin.ID); err != nil {
		t.Fatalf("admin GetIdeaForMember: %v", err)
	}

	// The RLS policy itself (not just the query filters): a member's
	// actor-context SELECT sees zero rows; the admin's sees the row.
	for _, tc := range []struct {
		name string
		user *User
		want int
	}{{"fellow member", other, 0}, {"workspace admin", admin, 1}} {
		var count int
		if err := st.withUserContext(ctx, tc.user.ID, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, `SELECT COUNT(*) FROM insideout.ideas WHERE id = $1`, idea.ID).Scan(&count)
		}); err != nil {
			t.Fatalf("count ideas as %s: %v", tc.name, err)
		}
		if count != tc.want {
			t.Fatalf("RLS %s sees %d rows, want %d", tc.name, count, tc.want)
		}
	}

	// The admin's drop authority still works on an idea they cannot author.
	if err := st.DropIdea(ctx, admin.ID, idea.ID); err != nil {
		t.Fatalf("admin DropIdea: %v", err)
	}
}
