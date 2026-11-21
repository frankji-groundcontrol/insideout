package agentcontext

import (
	"slices"
	"testing"
)

// A node's unfinished dependencies reach the briefing as waitsOn titles, on
// the focus and on each leaf; finished ones (done, validated) do not.
func TestTheBriefingListsUnfinishedDependenciesAsWaitsOn(t *testing.T) {
	nodes := []NodeIn{
		{ID: "root", Title: "Root", Status: "pending"},
		{ID: "a", ParentID: "root", Title: "Card", Status: "pending", DependsOn: []string{"b", "c", "d"}},
		{ID: "b", ParentID: "root", Title: "Ledger", Status: "in_progress"},
		{ID: "c", ParentID: "root", Title: "Schema", Status: "done"},
		{ID: "d", ParentID: "root", Title: "Audit", Status: "validated"},
	}
	out := Assemble(Inputs{Mode: "implementation", FocusNodeID: "a", Nodes: nodes})
	if got := out["focus"].(map[string]any)["waitsOn"]; !slices.Equal(got.([]string), []string{"Ledger"}) {
		t.Errorf("focus waitsOn = %v, want [Ledger]", got)
	}
	leaves := out["leaves"].([]map[string]any)
	if len(leaves) != 1 || !slices.Equal(leaves[0]["waitsOn"].([]string), []string{"Ledger"}) {
		t.Errorf("leaves = %v, want Card waiting on Ledger", leaves)
	}
	fields := out["untrusted"].(map[string]any)["fields"].([]string)
	for _, f := range []string{"focus.waitsOn[]", "leaves[].waitsOn[]"} {
		if !slices.Contains(fields, f) {
			t.Errorf("untrusted fields %v miss %s", fields, f)
		}
	}

	// A node that waits on nothing unfinished carries no waitsOn.
	out = Assemble(Inputs{Mode: "implementation", FocusNodeID: "b", Nodes: nodes})
	if _, has := out["focus"].(map[string]any)["waitsOn"]; has {
		t.Error("a node waiting on nothing has waitsOn")
	}
}
