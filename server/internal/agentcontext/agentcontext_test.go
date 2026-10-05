package agentcontext

import (
	"testing"
	"time"
)

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

func TestImplementationContextCarriesDeadlineAndPressure(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	deadline := now.Add(24 * time.Hour)
	out := Assemble(Inputs{
		Mode: "implementation", FocusNodeID: "a", Now: now,
		Nodes: []NodeIn{
			{ID: "root", Title: "交付 MVP", Status: "pending"},
			{ID: "a", ParentID: "root", Title: "实现 A", Status: "in_progress", Deadline: &deadline},
			{ID: "b", ParentID: "root", Title: "实现 B", Status: "pending"},
		},
	})
	focus := out["focus"].(map[string]any)
	if focus["deadline"] != deadline.Format(time.RFC3339) || focus["pressure"] != "high_risk" {
		t.Errorf("focus deadline/pressure = %v %v, want %s high_risk", focus["deadline"], focus["pressure"], deadline.Format(time.RFC3339))
	}
	leaves := out["leaves"].([]map[string]any)
	var leaf map[string]any
	for _, l := range leaves {
		if l["id"] == "a" {
			leaf = l
		}
	}
	if leaf == nil || leaf["deadline"] != deadline.Format(time.RFC3339) || leaf["pressure"] != "high_risk" {
		t.Errorf("leaf deadline/pressure missing: %v", leaves)
	}
	if _, has := focus["acceptance"]; has {
		t.Errorf("acceptance must stay absent until the column exists: %v", focus)
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
	// risks is blank, so the assembled open questions must carry at least
	// one blank-section entry — an empty list is the failure mode this
	// test exists to catch (TEST-05: it used to log and pass).
	if len(open) == 0 {
		t.Errorf("open questions = %v, want a blank-section entry (risks is blank)", open)
	} else if open[0] != "risks" && open[0] != "users" && open[0] != "nonGoals" && open[0] != "constraints" && open[0] != "requirements" && open[0] != "stories" {
		t.Errorf("open questions = %v, want a blank-section key first", open)
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
