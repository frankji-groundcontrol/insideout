package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// rv4-CLOSE-03: declining a proposed stage was a client-only state. A reload
// brought the card back, and the resolutions parked with it stayed parked for
// whichever confirm came next. Declining is now a write: the proposal and what
// was parked with it go together, and the stage and the findings stay as they were.

// decline posts the person's "not yet" for the stage the coach proposed.
func (f stageFixture) decline(t *testing.T, stage string) (int, trustView) {
	t.Helper()
	return f.declineAs(t, f.owner.ID, stage)
}

func (f stageFixture) declineAs(t *testing.T, userID uuid.UUID, stage string) (int, trustView) {
	t.Helper()
	token, err := f.tokens.MintAccessToken(userID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	resp := doJSON(t, f.ts, token, http.MethodPost, "/api/v1/conversations/"+f.convID.String()+"/stage/decline", `{"stage":"`+stage+`"}`)
	defer resp.Body.Close()
	var v trustView
	_ = json.NewDecoder(resp.Body).Decode(&v) // error bodies carry no view
	return resp.StatusCode, v
}

func TestDeclineStageClearsTheProposalAndWhatWasParkedWithIt(t *testing.T) {
	f := newStageFixture(t)
	f.parkMeta(t, parkedResolutionsMeta)

	status, v := f.decline(t, "draft")
	if status != http.StatusOK {
		t.Fatalf("decline: status %d, want 200", status)
	}
	if v.PendingStage != "" || len(v.PendingResolutions) != 0 {
		t.Fatalf("the answer still carries pendingStage %q and %d resolutions", v.PendingStage, len(v.PendingResolutions))
	}

	meta := f.meta(t)
	for _, key := range []string{"pendingStage", "pendingFindingResolutions"} {
		if _, kept := meta[key]; kept {
			t.Fatalf("meta still has %s after a decline: %s", key, meta[key])
		}
	}
	var findings []struct{ ID, Status string }
	if err := json.Unmarshal(meta["critic_findings"], &findings); err != nil {
		t.Fatalf("critic_findings: %v", err)
	}
	for _, fnd := range findings {
		if fnd.Status != "open" {
			t.Fatalf("finding %s is %s: a decline applies nothing", fnd.ID, fnd.Status)
		}
	}
	if string(meta["critic_state"]) == "" {
		t.Fatalf("critic_state was dropped with the proposal")
	}
	conv, err := f.st.GetConversationForOwner(context.Background(), f.convID, f.owner.ID)
	if err != nil {
		t.Fatalf("reload conversation: %v", err)
	}
	if conv.Stage != "clarify" {
		t.Fatalf("stage = %q, want the unadvanced clarify", conv.Stage)
	}
}

// A decline that only the page remembers is the bug: it has to survive a
// reload and leave nothing for a later confirm to pick up.
func TestDeclinedStageDoesNotComeBackOnReloadOrConfirm(t *testing.T) {
	f := newStageFixture(t)
	f.parkMeta(t, parkedResolutionsMeta)
	if status, _ := f.decline(t, "draft"); status != http.StatusOK {
		t.Fatalf("decline: status %d", status)
	}

	if v := f.view(t); v.PendingStage != "" || len(v.PendingResolutions) != 0 {
		t.Fatalf("a reload shows pendingStage %q and %d resolutions again", v.PendingStage, len(v.PendingResolutions))
	}
	if status := f.confirm(t, "draft"); status != http.StatusConflict {
		t.Fatalf("confirming the declined stage: status %d, want 409", status)
	}
}

func TestDeclineStageOnlyDeclinesTheStageThatIsPending(t *testing.T) {
	f := newStageFixture(t)
	f.parkMeta(t, parkedResolutionsMeta)

	if status, _ := f.decline(t, "finalize"); status != http.StatusConflict {
		t.Fatalf("declining a stage nobody proposed: status %d, want 409", status)
	}
	if v := f.view(t); v.PendingStage != "draft" || len(v.PendingResolutions) != 1 {
		t.Fatalf("a refused decline changed the proposal: pendingStage %q, %d resolutions", v.PendingStage, len(v.PendingResolutions))
	}

	if status, _ := f.decline(t, "draft"); status != http.StatusOK {
		t.Fatalf("decline: status %d", status)
	}
	if status, _ := f.decline(t, "draft"); status != http.StatusConflict {
		t.Fatalf("declining twice: status %d, want 409 (nothing is waiting)", status)
	}
}

func TestDeclineStageIsTheOwnersOnly(t *testing.T) {
	f := newStageFixture(t)
	f.parkMeta(t, parkedResolutionsMeta)
	other, err := f.st.CreateUser(context.Background(), "other-"+uuid.NewString()+"@example.com", "x", "other")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	if status, _ := f.declineAs(t, other.ID, "draft"); status != http.StatusConflict {
		t.Fatalf("another user's decline: status %d, want 409 like confirm gives", status)
	}
	if v := f.view(t); v.PendingStage != "draft" {
		t.Fatalf("another user cleared the owner's proposal: pendingStage %q", v.PendingStage)
	}
}

func TestDeclineStageNeedsAStage(t *testing.T) {
	f := newStageFixture(t)
	f.parkMeta(t, parkedResolutionsMeta)

	if status, _ := f.decline(t, "  "); status != http.StatusBadRequest {
		t.Fatalf("blank stage: status %d, want 400", status)
	}
}
