package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// rv4-LEAD-PROPOSE-VALIDATE: handleAgentPropose wrote the agent_proposal row
// first and checked the items afterwards, so a 400 left an orphan proposal in
// the timeline that a human could still accept. A refused propose must write
// nothing. Real PostgreSQL through the whole handler chain.

// proposalRows counts the agent_proposal rows in the world's project.
func (w *routeWorld) proposalRows(t *testing.T) int {
	t.Helper()
	updates, err := w.st.ListProjectUpdates(context.Background(), w.owner.ID, w.project.ID, 100, nil)
	if err != nil {
		t.Fatalf("list updates: %v", err)
	}
	n := 0
	for _, u := range updates {
		if u.Kind == "agent_proposal" {
			n++
		}
	}
	return n
}

func TestRefusedProposeLeavesNoOrphanProposal(t *testing.T) {
	elevenItems := make([]string, 11)
	for i := range elevenItems {
		elevenItems[i] = fmt.Sprintf(`{"title":"item %d"}`, i)
	}
	cases := []struct{ name, items string }{
		{"blank title", `[{"title":"   "}]`},
		{"title over 200 characters", `[{"title":"` + strings.Repeat("a", 201) + `"}]`},
		{"unknown action", `[{"action":"delete_node","title":"x"}]`},
		{"bad item after a good one", `[{"title":"fine"},{"title":""}]`},
		{"eleven items", "[" + strings.Join(elevenItems, ",") + "]"},
	}
	w := newRouteWorld(t)
	for _, c := range cases {
		status, body := w.call(w.owner, http.MethodPost, "/api/v1/agent/propose",
			`{"projectId":"`+w.project.ID.String()+`","kind":"structure","summary":"refused","items":`+c.items+`}`)
		expect(t, c.name, status, body, http.StatusBadRequest)
		if n := w.proposalRows(t); n != 0 {
			t.Fatalf("%s: the 400 left %d agent_proposal row(s) in the project timeline, want none", c.name, n)
		}
	}

	// The validation moved ahead of the write; a good proposal still lands.
	w.proposeAsOwner(t, "a good one")
	if n := w.proposalRows(t); n != 1 {
		t.Fatalf("after one valid propose the timeline has %d proposals, want 1", n)
	}
}
