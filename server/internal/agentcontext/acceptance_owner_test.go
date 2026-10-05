package agentcontext

import (
	"testing"
	"time"
)

// Gap slice 36: the implementation context an agent is handed carries what
// the product says it needs to know done — the acceptance sentence, the
// deadline, and the pressure — plus the owner.
func TestImplementationContextCarriesAcceptanceOwnerDeadline(t *testing.T) {
	deadline := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	focus := NodeIn{
		ID: "n1", Title: "Wire the flow", Status: "in_progress", Deadline: &deadline,
		Acceptance: "Done when /healthz answers 200", OwnerName: "dana",
	}
	leaf := NodeIn{ID: "n2", ParentID: "n1", Title: "Sketch copy", Status: "pending",
		Acceptance: "Done when the copy is approved", OwnerName: "sam"}

	got := Assemble(Inputs{
		Mode: "implementation", ProjectID: "p1", ProjectTitle: "Gap", FocusNodeID: "n1",
		Nodes:          []NodeIn{focus, leaf},
		EvidenceCounts: map[string]int{"n1": 2},
		Now:            time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC),
	})

	node, ok := got["focus"].(map[string]any)
	if !ok {
		t.Fatalf("no focus node in context: %+v", got)
	}
	if node["acceptance"] != "Done when /healthz answers 200" {
		t.Fatalf("focus acceptance = %v", node["acceptance"])
	}
	if node["owner"] != "dana" {
		t.Fatalf("focus owner = %v", node["owner"])
	}
	if node["deadline"] == nil || node["pressure"] == nil {
		t.Fatalf("focus deadline/pressure missing: %+v", node)
	}

	leaves, ok := got["leaves"].([]map[string]any)
	if !ok || len(leaves) != 1 {
		t.Fatalf("leaves = %+v", got["leaves"])
	}
	if leaves[0]["acceptance"] != "Done when the copy is approved" || leaves[0]["owner"] != "sam" {
		t.Fatalf("leaf acceptance/owner = %+v", leaves[0])
	}
}
