package agent

import (
	"context"
	"encoding/json"
	"testing"
)

// seedBlockingFindings stores open blocking findings in conversation meta
// with a critic round recorded — the state the critique stage is in when
// the coach starts calling resolve_finding.
func seedBlockingFindings(t *testing.T, ex *toolExecutor, findings ...CriticFinding) {
	t.Helper()
	ctx := context.Background()
	conv, err := ex.store.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	lm, extra := loadLedger(conv.Meta)
	saveCriticState(extra, criticState{RoundCount: 1})
	saveCriticFindings(extra, findings)
	meta, err := saveLedger(lm, extra)
	if err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if err := ex.store.UpdateConversationMeta(ctx, ex.actorID, ex.conversationID, meta); err != nil {
		t.Fatalf("store meta: %v", err)
	}
}

// AIDEV-07 (review 3): resolve_finding executed on the model's say-so —
// the executor validated only id and status, so a steered model could
// unilaterally mark every blocking finding resolved/overridden and then
// propose finalize; the only human checkpoint (confirming the stage) never
// saw what the model auto-closed. The stored status must not change on a
// model turn: the request parks in meta as pendingFindingResolutions and
// is applied by the human confirmation that confirms the stage.
func TestResolveFinding_WaitsForHumanConfirmation(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)
	seedBlockingFindings(t, ex, CriticFinding{
		ID: "c1", Section: "goals", Severity: "blocking", Kind: "omission",
		Issue: "no measurable objective", Status: FindingOpen,
	})

	if _, err := ex.resolveFinding(ctx, `{"id":"c1","status":"overridden"}`); err != nil {
		t.Fatalf("resolve_finding: %v", err)
	}

	held, err := st.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation after resolve: %v", err)
	}
	_, extra := loadLedger(held.Meta)

	findings := loadCriticFindings(extra)
	if len(findings) != 1 || findings[0].Status != FindingOpen {
		t.Fatalf("resolve_finding flipped the stored status on the model's say-so — it must stay %q until the human confirms: %+v", FindingOpen, findings)
	}

	var pending []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if raw, ok := extra["pendingFindingResolutions"]; ok {
		_ = json.Unmarshal(raw, &pending)
	}
	if len(pending) != 1 || pending[0].ID != "c1" || pending[0].Status != FindingOverridden {
		t.Fatalf("the requested resolution should be parked as pendingFindingResolutions for the human to confirm, got %+v", pending)
	}
}

// The parked resolution is what lets the finalize proposal through — the
// human confirms it in the same event that confirms the stage, so parking
// must not deadlock the legit path (a blocker with no parked resolution
// still blocks).
func TestStageGate_Finalize_ParkedVsUnparkedBlockers(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)
	seedBlockingFindings(t, ex,
		CriticFinding{ID: "c1", Section: "goals", Severity: "blocking", Kind: "omission", Issue: "no measurable objective", Status: FindingOpen},
		CriticFinding{ID: "c2", Section: "users", Severity: "blocking", Kind: "omission", Issue: "audience too broad", Status: FindingOpen},
	)
	ex.currentStage = StageCritique

	// One of the two blockers resolved: the other still blocks.
	if _, err := ex.resolveFinding(ctx, `{"id":"c1","status":"resolved"}`); err != nil {
		t.Fatalf("resolve_finding c1: %v", err)
	}
	if _, err := ex.advanceStage(ctx, `{"next":"finalize"}`); err == nil {
		t.Fatal("advance with an unparked open blocking finding (c2) should be denied, was allowed")
	}

	if _, err := ex.resolveFinding(ctx, `{"id":"c2","status":"overridden"}`); err != nil {
		t.Fatalf("resolve_finding c2: %v", err)
	}
	if _, err := ex.advanceStage(ctx, `{"next":"finalize"}`); err != nil {
		t.Fatalf("advance with every blocking finding's resolution parked for the user's confirmation should be allowed: %v", err)
	}
}

// A parked resolution is a proposal, not a fact: the stored finding keeps
// its open status (already asserted above), and a second call for the same
// finding replaces the parked pair instead of piling up duplicates.
func TestResolveFinding_ReParkReplacesSameID(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)
	seedBlockingFindings(t, ex, CriticFinding{
		ID: "c1", Section: "goals", Severity: "blocking", Kind: "omission",
		Issue: "no measurable objective", Status: FindingOpen,
	})

	if _, err := ex.resolveFinding(ctx, `{"id":"c1","status":"resolved"}`); err != nil {
		t.Fatalf("resolve_finding first: %v", err)
	}
	if _, err := ex.resolveFinding(ctx, `{"id":"c1","status":"overridden"}`); err != nil {
		t.Fatalf("resolve_finding second: %v", err)
	}

	held, err := st.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	_, extra := loadLedger(held.Meta)
	var pending []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if raw, ok := extra["pendingFindingResolutions"]; ok {
		_ = json.Unmarshal(raw, &pending)
	}
	if len(pending) != 1 || pending[0].Status != FindingOverridden {
		t.Fatalf("re-resolving should replace the parked pair, got %+v", pending)
	}
}
