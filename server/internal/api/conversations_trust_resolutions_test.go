package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

// rv4-AIDEV-03: resolve_finding only parks a status; the person's confirmation
// of the stage change is what applies it. The conversation view carried only
// pendingStage, so the confirm card could not show what a click would resolve
// or override. These tests pin the view to the same rules ApplyStageConfirmation
// applies: an unknown finding or a status confirm would not honour is not shown.

// cf1 is parked as overridden (the case a person most needs to see), ghost no
// longer exists, and cf2 carries a status confirm would skip.
const parkedResolutionsMeta = `{
	"critic_findings": [
		{"id":"cf1","section":"background","severity":"blocking","kind":"defect","issue":"no problem stated","status":"open"},
		{"id":"cf2","section":"goals","severity":"major","kind":"omission","issue":"no goal","status":"open"}
	],
	"critic_state": {"roundCount":1},
	"pendingFindingResolutions": [
		{"id":"cf1","status":"overridden"},
		{"id":"ghost","status":"resolved"},
		{"id":"cf2","status":"bogus"}
	],
	"pendingStage":"draft"
}`

func TestGetConversationExposesTheParkedFindingResolutions(t *testing.T) {
	f := newStageFixture(t)
	f.parkMeta(t, parkedResolutionsMeta)

	got := f.view(t).PendingResolutions
	if len(got) != 1 {
		t.Fatalf("pendingResolutions = %+v, want only cf1: ghost is gone and cf2's status would not be applied", got)
	}
	want := "cf1/background/blocking/no problem stated/overridden"
	r := got[0]
	if have := r.ID + "/" + r.Section + "/" + r.Severity + "/" + r.Issue + "/" + r.Status; have != want {
		t.Fatalf("pendingResolutions[0] = %q, want %q", have, want)
	}
}

func TestGetConversationHasNoResolutionsWhenNothingIsParked(t *testing.T) {
	f := newStageFixture(t)
	f.parkMeta(t, `{
		"critic_findings": [{"id":"cf1","section":"background","severity":"blocking","kind":"defect","issue":"no problem stated","status":"open"}],
		"pendingStage":"draft"
	}`)

	if got := f.view(t).PendingResolutions; len(got) != 0 {
		t.Fatalf("pendingResolutions = %+v, want none: nothing is parked", got)
	}
}

// The card is only honest if it shows exactly what the click applies.
func TestWhatTheViewShowsIsWhatConfirmApplies(t *testing.T) {
	f := newStageFixture(t)
	f.parkMeta(t, `{
		"critic_findings": [
			{"id":"cf1","section":"background","severity":"blocking","kind":"defect","issue":"no problem stated","status":"open"},
			{"id":"cf2","section":"goals","severity":"major","kind":"omission","issue":"no goal","status":"open"},
			{"id":"cf3","section":"risks","severity":"minor","kind":"omission","issue":"no risks","status":"open"}
		],
		"pendingFindingResolutions": [
			{"id":"cf1","status":"resolved"},
			{"id":"cf3","status":"overridden"}
		],
		"pendingStage":"draft"
	}`)

	shown := map[string]string{}
	for _, r := range f.view(t).PendingResolutions {
		shown[r.ID] = r.Status
	}

	if status := f.confirm(t, "draft"); status != http.StatusOK {
		t.Fatalf("confirm: status %d", status)
	}
	var findings []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(f.meta(t)["critic_findings"], &findings); err != nil {
		t.Fatalf("unmarshal findings: %v", err)
	}
	applied := map[string]string{}
	for _, fnd := range findings {
		if fnd.Status != "open" {
			applied[fnd.ID] = fnd.Status
		}
	}

	if len(shown) != 2 || len(applied) != 2 {
		t.Fatalf("shown %v, applied %v: want the same two findings in both", shown, applied)
	}
	for id, status := range applied {
		if shown[id] != status {
			t.Fatalf("confirm applied %s=%s but the view showed %q", id, status, shown[id])
		}
	}
}
