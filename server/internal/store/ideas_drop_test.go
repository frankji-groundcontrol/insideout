package store

import (
	"context"
	"testing"
)

// USER-09 (review 2): Drop must visibly drop. 'dropped' is a soft terminal
// state — the row stays for audit, but the author's wall is a working
// surface, not an archive: ListIdeas must stop returning dropped ideas the
// moment they are dropped. 'converted' ideas stay listed (they are the
// doorway to the PRD they became).
func TestIdeaDrop_DisappearsFromAuthorsWall(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	author := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Drop Visibility WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, author.ID, ws.Code); err != nil {
		t.Fatalf("join workspace: %v", err)
	}
	dropped, err := st.CreateIdea(ctx, author.ID, ws.ID, "Cold lead", "park it")
	if err != nil {
		t.Fatalf("create dropped-candidate idea: %v", err)
	}
	converted, err := st.CreateIdea(ctx, author.ID, ws.ID, "Hot lead", "build it")
	if err != nil {
		t.Fatalf("create converted-candidate idea: %v", err)
	}

	if _, _, err := st.ConvertIdea(ctx, author.ID, converted.ID); err != nil {
		t.Fatalf("convert idea: %v", err)
	}
	if err := st.DropIdea(ctx, author.ID, dropped.ID); err != nil {
		t.Fatalf("DropIdea: %v", err)
	}

	got, err := st.ListIdeas(ctx, author.ID, ws.ID)
	if err != nil {
		t.Fatalf("author ListIdeas after drop: %v", err)
	}
	var sawDropped, sawConverted bool
	for _, i := range got {
		if i.ID == dropped.ID {
			sawDropped = true
		}
		if i.ID == converted.ID {
			sawConverted = true
		}
	}
	if sawDropped {
		t.Fatal("dropped idea still listed on the author's wall — Drop never visibly drops (USER-09)")
	}
	if !sawConverted {
		t.Fatal("converted idea should stay listed — it is the doorway to its PRD")
	}
}
