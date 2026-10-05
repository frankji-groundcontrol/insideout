package roadmaptime

import (
	"testing"
	"time"
)

// Gap slices 16/17: every progress item carries the acceptance sentence and
// the owner's name, so the board says what done means and who holds it.
func TestAssemble_CarriesAcceptanceAndOwner(t *testing.T) {
	deadline := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	p := Assemble([]Node{
		{ID: "n1", Title: "Wire the flow", Status: "in_progress", Deadline: &deadline, Leaf: true,
			Acceptance: "Done when /healthz answers 200", OwnerName: "dana"},
		{ID: "n2", Title: "Sketch copy", Status: "done", Leaf: true,
			Acceptance: "Done when the copy is approved", OwnerName: "sam"},
	}, now)

	if len(p.Now) != 1 {
		t.Fatalf("Now has %d items, want 1", len(p.Now))
	}
	got := p.Now[0]
	if got.Acceptance != "Done when /healthz answers 200" {
		t.Fatalf("Now item Acceptance = %q", got.Acceptance)
	}
	if got.OwnerName != "dana" {
		t.Fatalf("Now item OwnerName = %q", got.OwnerName)
	}
	done := p.DoneItems
	if len(done) != 1 || done[0].OwnerName != "sam" || done[0].Acceptance == "" {
		t.Fatalf("DoneItems = %+v, want sam's item with its acceptance", done)
	}
}
