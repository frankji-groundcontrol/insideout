package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

// trustView is the part of GET /conversations/{id} the coach trust layer adds:
// what the confirm card needs to be honest about (rv4-AIDEV-02, rv4-AIDEV-03).
type trustView struct {
	PendingStage       string `json:"pendingStage"`
	CriticSkipped      string `json:"criticSkipped"`
	CriticRounds       int    `json:"criticRounds"`
	PendingResolutions []struct {
		ID       string `json:"id"`
		Section  string `json:"section"`
		Severity string `json:"severity"`
		Issue    string `json:"issue"`
		Status   string `json:"status"`
	} `json:"pendingResolutions"`
}

func (f stageFixture) view(t *testing.T) trustView {
	t.Helper()
	token, err := f.tokens.MintAccessToken(f.owner.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	resp := doJSON(t, f.ts, token, http.MethodGet, "/api/v1/conversations/"+f.convID.String(), "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET conversation: status %d", resp.StatusCode)
	}
	var v trustView
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode conversation: %v", err)
	}
	return v
}

// A critic pass that did not run opens the finalize gate; the person confirming
// the finalize must be able to see that nothing was reviewed.
func TestGetConversationExposesACriticSkip(t *testing.T) {
	f := newStageFixture(t)
	f.parkMeta(t, `{"critic_state":{"roundCount":0,"skipped":"contention"},"pendingStage":"finalize"}`)

	v := f.view(t)
	if v.CriticSkipped != "contention" {
		t.Fatalf("criticSkipped = %q, want the stored reason contention", v.CriticSkipped)
	}
	if v.CriticRounds != 0 {
		t.Fatalf("criticRounds = %d, want 0: no critic review ran", v.CriticRounds)
	}
}

func TestGetConversationHasNoCriticSkipAfterAReviewedRound(t *testing.T) {
	f := newStageFixture(t)
	f.parkMeta(t, `{"critic_state":{"roundCount":1},"pendingStage":"finalize"}`)

	v := f.view(t)
	if v.CriticSkipped != "" {
		t.Fatalf("criticSkipped = %q on a reviewed PRD, want none", v.CriticSkipped)
	}
	if v.CriticRounds != 1 {
		t.Fatalf("criticRounds = %d, want the 1 round that ran", v.CriticRounds)
	}
}
