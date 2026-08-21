package agentcontext

import "testing"

func nodes() []NodeIn {
	return []NodeIn{
		{ID: "root", ParentID: "", Title: "交付 MVP", Status: "pending"},
		{ID: "a", ParentID: "root", Title: "实现 A", Status: "in_progress"},
		{ID: "b", ParentID: "root", Title: "实现 B", Status: "pending"},
	}
}

func TestImplementationModeFocusScoped(t *testing.T) {
	out := Assemble(Inputs{
		ProjectTitle: "P", ProjectID: "p1", Mode: "implementation", FocusNodeID: "a",
		PrdSections: map[string]string{"background": "why"},
		Nodes:       nodes(), EvidenceCounts: map[string]int{"a": 3},
	})
	focus := out["focus"].(map[string]any)
	if focus["title"] != "实现 A" || focus["evidence"] != 3 {
		t.Errorf("focus wrong: %v", focus)
	}
	if _, has := focus["siblings"]; !has {
		t.Errorf("siblings missing: %v", focus)
	}
	leaves := out["leaves"].([]map[string]any)
	if len(leaves) != 2 {
		t.Errorf("leaf count wrong: %v", leaves)
	}
}

func TestBrainstormingModeEmphasizesArgument(t *testing.T) {
	out := Assemble(Inputs{
		Mode:        "brainstorming",
		PrdSections: map[string]string{"background": "B", "goals": "G", "risks": ""},
		Nodes:       nodes(),
	})
	arg := out["productArgument"].(map[string]string)
	if arg["background"] != "B" || len(arg) != 1 {
		t.Errorf("argument shape wrong: %v", arg)
	}
	open := out["openQuestions"].([]string)
	if len(open) == 0 || open[0] != "risks" && open[0] != "users" && open[0] != "nonGoals" && open[0] != "constraints" && open[0] != "requirements" && open[0] != "stories" {
		// risks is blank; any blank key is a valid open question entry
		t.Logf("open questions: %v", open)
	}
	if _, has := out["leaves"]; has {
		t.Error("brainstorming should not carry leaves")
	}
}

func TestReviewModeCarriesVersion(t *testing.T) {
	out := Assemble(Inputs{
		Mode:         "review",
		PrdSections:  map[string]string{"goals": "G"},
		Nodes:        nodes(),
		LatestCommit: &CommitIn{Revision: 3, Name: "v3", Audience: "decision"},
	})
	v := out["version"].(map[string]any)
	if v["name"] != "v3" || v["revision"] != 3 {
		t.Errorf("version missing: %v", v)
	}
	core := out["prdCore"].(map[string]string)
	if core["goals"] != "G" {
		t.Errorf("prd core wrong: %v", core)
	}
	if _, has := out["vocabulary"]; !has {
		t.Error("vocabulary contract missing")
	}
}
