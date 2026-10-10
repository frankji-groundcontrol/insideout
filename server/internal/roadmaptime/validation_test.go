package roadmaptime

import (
	"testing"
	"time"
)

// FEAT-04 (review 3): a validated leaf is finished work with a human
// confirm on top — the Done list keeps it, or the board would lose the
// row the moment a node validates.
func TestAssembleKeepsValidatedLeavesInDone(t *testing.T) {
	now := time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)
	p := Assemble([]Node{
		{ID: "v", Title: "validated leaf", Status: "validated", Leaf: true},
		{ID: "d", Title: "done leaf", Status: "done", Leaf: true},
	}, now)
	if p.DoneCount != 2 {
		t.Fatalf("DoneCount = %d, want 2 (validated counts as finished)", p.DoneCount)
	}
	if len(p.DoneItems) != 2 {
		t.Fatalf("DoneItems = %d, want 2", len(p.DoneItems))
	}
	var statuses []string
	for _, it := range p.DoneItems {
		statuses = append(statuses, it.Status)
	}
	if statuses[0] != "validated" || statuses[1] != "done" {
		t.Fatalf("DoneItems statuses = %v, want the item's own status kept", statuses)
	}
}
