package store

import (
	"context"
	"os"
	"strings"
	"testing"
)

// rv4-FEAT-09: the ProposalDecision doc called the Decision Log entry
// "immutable; re-deciding appends", but DecideProposal upserts the row: the row
// holds the latest decision and only the project timeline keeps history, one
// note per decide. The doc is what a reader trusts, so pin the behavior and
// require the doc to say it.
func TestProposalDecision_RowIsTheLatestAndTheTimelineAppends(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	f := newApplyFixture(t, st)

	if _, err := st.DecideProposal(ctx, f.owner.ID, f.upd.ID, "accepted", "first thought"); err != nil {
		t.Fatalf("accept: %v", err)
	}
	if _, err := st.DecideProposal(ctx, f.owner.ID, f.upd.ID, "rejected", "second thought"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	updates, err := st.ListProjectUpdates(ctx, f.owner.ID, f.proj.ID, 100, nil)
	if err != nil {
		t.Fatalf("list updates: %v", err)
	}
	var latest string
	var notes []string
	for _, u := range updates {
		if u.ID == f.upd.ID {
			latest = u.Decision
		}
		if u.Kind == "note" && strings.HasPrefix(u.Content, "[proposal ") {
			notes = append(notes, firstLine(u.Content))
		}
	}
	if latest != "rejected" {
		t.Errorf("after accepted then rejected the decision row holds %q, want the latest, rejected", latest)
	}
	if len(notes) != 2 {
		t.Errorf("the timeline has %d decision notes, want 2 (one per decide): %q", len(notes), notes)
	}

	src, err := os.ReadFile("proposal_decisions.go")
	if err != nil {
		t.Fatalf("read proposal_decisions.go: %v", err)
	}
	if strings.Contains(string(src), "immutable") {
		t.Errorf("proposal_decisions.go still calls the decision row immutable; DecideProposal overwrites it, only the timeline notes append")
	}
	if !strings.Contains(string(src), "latest") {
		t.Errorf("proposal_decisions.go does not say the row holds the latest decision")
	}
}
