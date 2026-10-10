package agent

import (
	"encoding/json"
	"testing"
)

func TestApplyStageConfirmation(t *testing.T) {
	raw := json.RawMessage(`{
		"facts": [],
		"critic_findings": [
			{"id":"cf1","section":"background","severity":"blocking","kind":"defect","issue":"a","status":"open"},
			{"id":"cf2","section":"goals","severity":"major","kind":"omission","issue":"b","status":"open"}
		],
		"pendingFindingResolutions": [
			{"id":"cf1","status":"resolved"},
			{"id":"gone","status":"overridden"},
			{"id":"cf2","status":"deleted"}
		],
		"pendingStage":"draft"
	}`)
	out, err := ApplyStageConfirmation(raw)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var findings []CriticFinding
	if err := json.Unmarshal(m["critic_findings"], &findings); err != nil {
		t.Fatalf("findings: %v", err)
	}
	byID := map[string]string{}
	for _, f := range findings {
		byID[f.ID] = f.Status
	}
	if byID["cf1"] != FindingResolved {
		t.Fatalf("cf1 = %q, want resolved", byID["cf1"])
	}
	if byID["cf2"] != FindingOpen {
		t.Fatalf("cf2 = %q, want open — an invalid parked status is not honored", byID["cf2"])
	}
	if _, parked := m["pendingFindingResolutions"]; parked {
		t.Fatal("pendingFindingResolutions survived the confirmation")
	}
	if _, proposal := m["pendingStage"]; proposal {
		t.Fatal("pendingStage survived the confirmation")
	}
	var trail []resolutionTrailEntry
	if err := json.Unmarshal(m["findingResolutionsApplied"], &trail); err != nil {
		t.Fatalf("trail: %v", err)
	}
	if len(trail) != 1 || trail[0].ID != "cf1" || trail[0].Status != FindingResolved || trail[0].At == "" {
		t.Fatalf("trail = %+v, want exactly the cf1 ratification with a timestamp", trail)
	}
}

func TestApplyStageConfirmationWithoutParkedWork(t *testing.T) {
	out, err := ApplyStageConfirmation(json.RawMessage(`{"pendingStage":"critique"}`))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(m) != 0 {
		t.Fatalf("meta = %s, want the consumed proposal and nothing else", out)
	}
	if _, trail := m["findingResolutionsApplied"]; trail {
		t.Fatal("an empty confirmation must not write a trail entry")
	}
}
