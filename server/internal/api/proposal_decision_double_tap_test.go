package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// rv4-FR-F02, F08, F15 (server half): a double tap on Accept sends the same
// decision twice. The nodes were already built once; the history note was
// appended every time. The route answers both copies 200 and the timeline
// records one decision, while the opposite decision stays a decision of its own.
func TestRvStoreDbDecision_ADoubleTapAddsOneNote(t *testing.T) {
	w := newRouteWorld(t)
	id := w.proposeItems(t, "Alpha "+uuid.NewString()[:8])
	notes := func() []string {
		t.Helper()
		updates, err := w.st.ListProjectUpdates(context.Background(), w.owner.ID, w.project.ID, 100, nil)
		if err != nil {
			t.Fatalf("list updates: %v", err)
		}
		var out []string
		for _, u := range updates {
			if u.Kind == "note" && strings.HasPrefix(u.Content, "[proposal ") {
				out = append(out, u.Content)
			}
		}
		return out
	}

	for tap := 1; tap <= 2; tap++ {
		status, body := w.decide(w.owner, id, acceptAndApply)
		expect(t, "accept", status, body, http.StatusOK)
	}
	if got := notes(); len(got) != 1 {
		t.Errorf("two taps on Accept left %d decision notes, want 1: %q", len(got), got)
	}
	status, body := w.decide(w.owner, id, `{"decision":"rejected"}`)
	expect(t, "reject", status, body, http.StatusOK)
	if got := notes(); len(got) != 2 {
		t.Errorf("accepting then rejecting left %d decision notes, want 2: %q", len(got), got)
	}
	if d := w.decisionOf(t, id); d != "rejected" {
		t.Errorf("the record says %q after the reversal, want rejected (a decision may be reversed)", d)
	}
}
