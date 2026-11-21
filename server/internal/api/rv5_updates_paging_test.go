package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// rv5 row 3.3 (PARB-07): the project page embeds the newest 50 updates and a
// nextCursor; GET /projects/{pid}/updates?before= takes that cursor and pages
// back through the rest. Real PostgreSQL.

type updatesPage struct {
	Updates []struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	} `json:"updates"`
	NextCursor *string `json:"nextCursor"`
}

func TestTheUpdatesFeedPagesBackThroughOlderUpdates(t *testing.T) {
	w := newCoverageWorld(t)
	ctx := context.Background()
	for i := 0; i < 55; i++ {
		if _, err := w.st.AddProjectUpdate(ctx, w.owner.ID, w.project.ID, "progress", fmt.Sprintf("note %d", i)); err != nil {
			t.Fatalf("note %d: %v", i, err)
		}
	}

	first := decodeAs[updatesPage](t, w.must(t, w.member, "GET", w.proj(), "", http.StatusOK))
	if len(first.Updates) != 50 || first.NextCursor == nil {
		t.Fatalf("project page: %d updates, cursor %v; want 50 and a cursor", len(first.Updates), first.NextCursor)
	}
	feed := w.proj() + "/updates"
	second := decodeAs[updatesPage](t, w.must(t, w.member, "GET", feed+"?before="+*first.NextCursor, "", http.StatusOK))
	if len(second.Updates) != 5 || second.NextCursor != nil {
		t.Fatalf("second page: %d updates, cursor %v; want 5 and no cursor", len(second.Updates), second.NextCursor)
	}
	seen := map[string]bool{}
	for _, u := range append(first.Updates, second.Updates...) {
		if seen[u.ID] {
			t.Fatalf("update %s is on both pages", u.ID)
		}
		seen[u.ID] = true
	}
	if second.Updates[4].Content != "note 0" {
		t.Errorf("oldest update = %q, want note 0", second.Updates[4].Content)
	}

	// The feed's own first page and a smaller limit.
	head := decodeAs[updatesPage](t, w.must(t, w.member, "GET", feed+"?limit=20", "", http.StatusOK))
	if len(head.Updates) != 20 || head.NextCursor == nil || head.Updates[0].Content != "note 54" {
		t.Fatalf("limit 20: %d updates, cursor %v, newest %q", len(head.Updates), head.NextCursor, head.Updates[0].Content)
	}
	w.must(t, w.asAgent(t, w.member), "GET", feed, "", http.StatusOK)

	for _, q := range []string{"?limit=0", "?limit=101", "?limit=x", "?before=not-a-cursor"} {
		w.must(t, w.member, "GET", feed+q, "", http.StatusBadRequest)
	}
	w.must(t, w.outsider, "GET", feed, "", http.StatusNotFound)
}
