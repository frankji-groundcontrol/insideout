package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// Feed scoping (2026-11-22 plan, S2): ?node= keeps the updates about a node
// and the nodes below it (node rows and proposals carry a node; notes,
// progress and blockers carry none), ?pending=true the proposals nobody has
// decided yet, for a caller who may decide them. Real PostgreSQL.

type feedRow struct {
	ID       string  `json:"id"`
	Kind     string  `json:"kind"`
	NodeID   *string `json:"nodeId"`
	Decision string  `json:"decision"`
}

func (w *sparkRoutes) feed(t *testing.T, as routeUser, query string, want int) []feedRow {
	t.Helper()
	body := w.must(t, as, "GET", "/api/v1/projects/"+w.spark.Project.ID.String()+"/updates"+query, "", want)
	if want != http.StatusOK {
		return nil
	}
	return decodeAs[struct {
		Updates []feedRow `json:"updates"`
	}](t, body).Updates
}

func TestTheFeedNarrowsToANodeAndToPendingProposals(t *testing.T) {
	w := newSparkRoutes(t)
	ctx := context.Background()
	engine, queue := w.spark.Node(t, "Prompt-based matching engine"), w.spark.Node(t, "Moderator review queue")
	decided, err := w.st.AddProposal(ctx, w.sam.ID, w.spark.Project.ID, &engine.ID, "Add a cold-start fallback.", nil)
	if err != nil {
		t.Fatalf("Sam proposes: %v", err)
	}
	if _, err := w.st.DecideProposal(ctx, w.maya.ID, decided.ID, "rejected", "Not for the beta."); err != nil {
		t.Fatalf("Maya decides: %v", err)
	}
	open, err := w.st.AddProposal(ctx, w.jo.ID, w.spark.Project.ID, &queue.ID, "Add two more moderators.", nil)
	if err != nil {
		t.Fatalf("Jo proposes: %v", err)
	}
	note, err := w.st.AddProjectUpdate(ctx, w.leo.ID, w.spark.Project.ID, "note", "Matching is on track.")
	if err != nil {
		t.Fatalf("note: %v", err)
	}

	matching := w.spark.Node(t, "Matching")
	below, err := w.st.ListRoadmapSubtree(ctx, w.leo.ID, w.spark.Project.ID, matching.ID)
	if err != nil {
		t.Fatalf("subtree: %v", err)
	}
	inside := map[string]bool{}
	for _, n := range below {
		inside[n.ID.String()] = true
	}
	rows := w.feed(t, w.leo, "?node="+matching.ID.String(), http.StatusOK)
	seen := map[string]bool{}
	for _, r := range rows {
		seen[r.ID] = true
		if r.NodeID == nil || !inside[*r.NodeID] {
			t.Errorf("?node=Matching returned %+v, about a node outside it", r)
		}
	}
	if !seen[decided.ID.String()] || seen[open.ID.String()] || seen[note.ID.String()] || len(rows) < 5 {
		t.Errorf("?node=Matching: %d rows, Sam's proposal %v, Jo's %v, the note %v", len(rows),
			seen[decided.ID.String()], seen[open.ID.String()], seen[note.ID.String()])
	}

	for _, who := range []routeUser{w.maya, w.jo} {
		if got := w.feed(t, who, "?pending=true", http.StatusOK); len(got) != 1 || got[0].ID != open.ID.String() {
			t.Errorf("%s: pending = %+v, want Jo's open proposal only", who.Username, got)
		}
	}
	if got := w.feed(t, w.leo, "?pending=true", http.StatusOK); len(got) != 0 {
		t.Errorf("Leo decides nothing: pending = %+v", got)
	}
	if got := w.feed(t, w.maya, "?pending=true&node="+matching.ID.String(), http.StatusOK); len(got) != 0 {
		t.Errorf("nothing open under Matching: %+v", got)
	}
	w.feed(t, w.maya, "?node="+w.root(t).String(), http.StatusNotFound)
	w.feed(t, w.maya, "?node=nope", http.StatusBadRequest)
	w.feed(t, w.maya, "?pending=maybe", http.StatusBadRequest)
}

// root is a node of another project, which ?node= must not accept.
func (w *sparkRoutes) root(t *testing.T) uuid.UUID {
	t.Helper()
	n, err := w.st.CreateRoadmapNode(context.Background(), w.owner.ID, w.project.ID, nil, "Elsewhere", "")
	if err != nil {
		t.Fatalf("node elsewhere: %v", err)
	}
	return n.ID
}
