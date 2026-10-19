package store

import (
	"context"
	"os"
	"strings"
	"sync"
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

// rv4-FR-F02, F08, F15 (server half): the web feed has no in-flight guard, so a
// double tap, a retry or a second tab sends the same decision twice, and every
// copy appended another "[proposal accepted]" note to the history all members
// read (the apply was already once-only). Deciding what is already decided,
// with the same reason, is not an event; a different decision or reason is.

// rvStoreDbDecisionNotes is the project's decision notes, newest first.
func rvStoreDbDecisionNotes(t *testing.T, st *Store, f *applyFixture) []string {
	t.Helper()
	updates, err := st.ListProjectUpdates(context.Background(), f.owner.ID, f.proj.ID, 100, nil)
	if err != nil {
		t.Fatalf("list updates: %v", err)
	}
	var notes []string
	for _, u := range updates {
		if u.Kind == "note" && strings.HasPrefix(u.Content, "[proposal ") {
			notes = append(notes, firstLine(u.Content))
		}
	}
	return notes
}

func TestRvStoreDbDecision_RepeatingADecisionIsNotAnEvent(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	f := newApplyFixture(t, st)

	first, err := st.DecideProposal(ctx, f.owner.ID, f.upd.ID, "accepted", "fits the plan")
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	again, err := st.DecideProposal(ctx, f.owner.ID, f.upd.ID, "accepted", "fits the plan")
	if err != nil {
		t.Fatalf("accept again: %v", err)
	}
	if !again.DecidedAt.Equal(first.DecidedAt) || again.DecidedBy != first.DecidedBy {
		t.Errorf("the repeat rewrote the record: decided %v by %v, then %v by %v",
			first.DecidedAt, first.DecidedBy, again.DecidedAt, again.DecidedBy)
	}
	if notes := rvStoreDbDecisionNotes(t, st, f); len(notes) != 1 {
		t.Errorf("a repeated accept left %d decision notes, want 1: %q", len(notes), notes)
	}

	// A new reason, the opposite decision, and the reversal of the reversal are
	// changes: each is recorded, as DecideProposal documents.
	for i, c := range []struct{ decision, reason string }{
		{"accepted", "fits the plan, but later"}, {"rejected", "changed my mind"}, {"accepted", "changed it back"},
	} {
		d, err := st.DecideProposal(ctx, f.owner.ID, f.upd.ID, c.decision, c.reason)
		if err != nil {
			t.Fatalf("%s: %v", c.decision, err)
		}
		if d.Decision != c.decision || d.Reason != c.reason {
			t.Errorf("after %q (%q) the record says %q (%q)", c.decision, c.reason, d.Decision, d.Reason)
		}
		if notes := rvStoreDbDecisionNotes(t, st, f); len(notes) != 2+i {
			t.Errorf("after %q (%q) the timeline has %d decision notes, want %d: %q", c.decision, c.reason, len(notes), 2+i, notes)
		}
	}
}

func TestRvStoreDbDecision_AcceptingAgainStillAppliesWhatIsLeft(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	f := newApplyFixture(t, st, ProposalItem{Title: "Intake"}, ProposalItem{Title: "Handoff"})

	first, ids, err := st.DecideAndApplyProposal(ctx, f.owner.ID, f.upd.ID, "accepted", "", false)
	if err != nil || len(ids) != 0 {
		t.Fatalf("accept without apply: %d nodes, err %v", len(ids), err)
	}
	second, ids, err := st.DecideAndApplyProposal(ctx, f.owner.ID, f.upd.ID, "accepted", "", true)
	if err != nil || len(ids) != 2 {
		t.Fatalf("accept with apply after accepting without: %d nodes, err %v, want 2: applying later is how an accepted proposal gets built", len(ids), err)
	}
	if !second.DecidedAt.Equal(first.DecidedAt) {
		t.Errorf("accepting again moved decided_at from %v to %v", first.DecidedAt, second.DecidedAt)
	}
	if notes := rvStoreDbDecisionNotes(t, st, f); len(notes) != 1 {
		t.Errorf("accepting twice left %d decision notes, want 1: %q", len(notes), notes)
	}
	f.wantTitles(t, st, "Intake", "Handoff")
}

func TestRvStoreDbDecision_TapsAtOnceLeaveOneNote(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	f := newApplyFixture(t, st, ProposalItem{Title: "Intake"}, ProposalItem{Title: "Handoff"})

	const taps = 4
	var wg sync.WaitGroup
	gate, built := make(chan struct{}), make(chan int, taps)
	for i := 0; i < taps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-gate
			_, ids, err := st.DecideAndApplyProposal(ctx, f.owner.ID, f.upd.ID, "accepted", "", true)
			if err != nil {
				t.Errorf("a tap failed: %v", err)
			}
			built <- len(ids)
		}()
	}
	close(gate)
	wg.Wait()
	close(built)
	total := 0
	for n := range built {
		total += n
	}
	if total != 2 {
		t.Errorf("%d taps built %d nodes in total, want 2", taps, total)
	}
	if notes := rvStoreDbDecisionNotes(t, st, f); len(notes) != 1 {
		t.Errorf("%d taps at once left %d decision notes, want 1: %q", taps, len(notes), notes)
	}
}
