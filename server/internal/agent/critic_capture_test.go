package agent

import (
	"context"
	"net/http/httptest"
	"testing"
)

// AIDEV-01 (review 3): a later critique-stage reply that parses as the
// ranked list wholesale-replaced critic_findings — the coach's restatement
// could erase the independent critic's open blocking finding and reset
// recorded resolutions, so the finalize gate read state the model alone
// could rewrite. A restatement must merge: keep unmatched stored findings
// (id + status intact), keep matched ones with their recorded status, and
// only add genuinely new lines.
func TestCaptureRankedCritique_RestatementCannotEraseStoredFindings(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)

	conv, err := st.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	lm, extra := loadLedger(conv.Meta)
	saveCriticState(extra, criticState{RoundCount: 1})
	saveCriticFindings(extra, []CriticFinding{
		{ID: "c1", Section: "goals", Severity: "blocking", Kind: "omission", Issue: "no measurable objective", Status: FindingOpen},
		{ID: "c2", Section: "users", Severity: "minor", Kind: "omission", Issue: "audience too broad", Status: FindingResolved},
	})
	meta, err := saveLedger(lm, extra)
	if err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if err := st.UpdateConversationMeta(ctx, ex.actorID, ex.conversationID, meta); err != nil {
		t.Fatalf("store meta: %v", err)
	}

	// The coach restates the list but omits the blocking finding and
	// renumbers the resolved one; one genuinely new line rides along.
	c := New(st, &forcingStub{}, discardLog())
	sse, ok := newSSEWriter(httptest.NewRecorder())
	if !ok {
		t.Fatal("recorder cannot flush")
	}
	c.captureRankedCritique(ctx, ex.actorID, ex.conversationID, StageCritique,
		"1. [major] risks — no risks listed\n2. [minor] users — audience too broad", sse)

	held, err := st.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation after capture: %v", err)
	}
	_, extra = loadLedger(held.Meta)
	findings := loadCriticFindings(extra)

	byID := map[string]CriticFinding{}
	for _, f := range findings {
		byID[f.ID] = f
	}
	// The failure AIDEV-01 describes: the omitted open blocking finding
	// must still be stored — hasOpenBlockingFindings reads exactly this
	// list (tools.go finalize gate).
	c1, kept := byID["c1"]
	if !kept || c1.Status != FindingOpen {
		t.Fatalf("open blocking finding c1 was erased or altered by the restatement: %+v", findings)
	}
	// A restated line keeps the stored id and its recorded status.
	c2, kept := byID["c2"]
	if !kept || c2.Status != FindingResolved {
		t.Fatalf("resolved finding c2 lost its recorded status through the restatement: %+v", findings)
	}
	// A genuinely new line is added, open, with its own id.
	var added bool
	for _, f := range findings {
		if f.Section == "risks" && f.Severity == "major" && f.Status == FindingOpen && f.ID != "" {
			added = true
		}
	}
	if !added {
		t.Fatalf("genuinely new restated line was not added: %+v", findings)
	}
}
