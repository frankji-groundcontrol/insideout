package agent

import (
	"context"
	"os"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// newTestStore connects to DATABASE_URL and skips if unset — same
// no-mocks convention as internal/store's integration tests
// (DATABASE_URL=... go test ./internal/agent/ -run TestStageGate -v).
func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	return st
}

// seedConversation creates a user, workspace, idea, and converts it to a
// PRD + conversation — the minimum fixture a toolExecutor needs.
func seedConversation(t *testing.T, st *store.Store) (userID string, executor *toolExecutor) {
	t.Helper()
	ctx := context.Background()

	u, err := st.CreateUser(ctx, "gates-"+uuid.NewString()+"@test.local", "x", "gates-tester")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, u.ID, "Gates Test WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, u.ID, ws.ID, "an idea", "content")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	prd, conv, err := st.ConvertIdea(ctx, u.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}

	return u.ID.String(), &toolExecutor{
		store: st, actorID: u.ID, prdID: prd.ID, conversationID: conv.ID, currentStage: StageClarify,
	}
}

func TestStageGate_ClarifyToDraft_RequiresAttestedFacts(t *testing.T) {
	st := newTestStore(t)
	_, ex := seedConversation(t, st)
	ctx := context.Background()

	if _, err := ex.advanceStage(ctx, `{"next":"draft"}`); err == nil {
		t.Fatal("advance clarify->draft with zero facts should be denied, was allowed")
	}

	ex.userMessages = []string{
		"honestly nobody can find their old notes once the workshop moves on",
		"today they just scroll through a giant Slack thread, which is terrible",
		"our workshop cohort of 15 designers, they're the ones stuck with this",
		"we need this before the next cohort starts in two weeks",
	}
	must := func(kind, text, quote string) {
		t.Helper()
		args := `{"kind":"` + kind + `","text":"` + text + `","quote":"` + quote + `"}`
		if _, err := ex.recordFact(ctx, args); err != nil {
			t.Fatalf("record_fact %s: %v", kind, err)
		}
	}
	must("problem", "users lose track of old notes", "nobody can find their old notes once the workshop moves on")
	must("segment", "a 15-person designer workshop cohort", "our workshop cohort of 15 designers, they're the ones stuck with this")
	must("alternative", "scrolling a Slack thread", "today they just scroll through a giant Slack thread, which is terrible")
	must("whynow", "next cohort starts in two weeks", "we need this before the next cohort starts in two weeks")

	if _, err := ex.advanceStage(ctx, `{"next":"draft"}`); err != nil {
		t.Fatalf("advance clarify->draft with all 4 facts attested: %v", err)
	}
}

func TestStageGate_ClarifyToDraft_RejectsEveryoneSegment(t *testing.T) {
	st := newTestStore(t)
	_, ex := seedConversation(t, st)
	ctx := context.Background()
	ex.userMessages = []string{
		"the problem is people forget stuff",
		"literally everyone has this problem, it's for all users",
		"they just forget, there's no real alternative today",
		"we want it now",
	}
	must := func(kind, text, quote string) {
		t.Helper()
		args := `{"kind":"` + kind + `","text":"` + text + `","quote":"` + quote + `"}`
		if _, err := ex.recordFact(ctx, args); err != nil {
			t.Fatalf("record_fact %s: %v", kind, err)
		}
	}
	must("problem", "people forget stuff", "the problem is people forget stuff")
	must("segment", "everyone, all users", "literally everyone has this problem, it's for all users")
	must("alternative", "nothing today", "they just forget, there's no real alternative today")
	must("whynow", "wanted now", "we want it now")

	if _, err := ex.advanceStage(ctx, `{"next":"draft"}`); err == nil {
		t.Fatal("advance with an \"everyone\" segment should be denied by the lint, was allowed")
	}
}

func TestStageGate_DraftToCritique_RequiresAllSectionsNonEmpty(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)
	ex.currentStage = StageDraft

	if _, err := ex.advanceStage(ctx, `{"next":"critique"}`); err == nil {
		t.Fatal("advance draft->critique with all sections empty should be denied, was allowed")
	}

	for _, section := range store.PrdSectionKeys {
		if _, err := ex.updateSection(ctx, `{"section":"`+section+`","markdown":"content for `+section+`"}`); err != nil {
			t.Fatalf("update section %s: %v", section, err)
		}
	}
	if _, err := ex.advanceStage(ctx, `{"next":"critique"}`); err != nil {
		t.Fatalf("advance draft->critique with all sections filled: %v", err)
	}
}

func TestRecordFact_RejectsUngroundedQuote(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)
	ex.userMessages = []string{"we lose a few customers here and there"}

	_, err := ex.recordFact(ctx, `{"kind":"problem","text":"80% of users churn weekly","quote":"80% of users churn every single week"}`)
	if err == nil {
		t.Fatal("record_fact with a quote the user never said should be rejected, was allowed")
	}
}

func TestStageGate_CritiqueToFinalize_RequiresCriticPassOrSkipMarker(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)
	ex.currentStage = StageCritique

	if _, err := ex.advanceStage(ctx, `{"next":"finalize"}`); err == nil {
		t.Fatal("advance critique->finalize with no critic pass and no skip marker should be denied, was allowed")
	}

	// Simulate the coach.go degradation path: record a skip marker
	// without ever running a real critic pass.
	conv, err := st.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	lm, extra := loadLedger(conv.Meta)
	saveCriticState(extra, criticState{Skipped: "contention"})
	metaJSON, err := saveLedger(lm, extra)
	if err != nil {
		t.Fatalf("save ledger: %v", err)
	}
	if err := st.UpdateConversationMeta(ctx, ex.actorID, ex.conversationID, metaJSON); err != nil {
		t.Fatalf("update conversation meta: %v", err)
	}

	if _, err := ex.advanceStage(ctx, `{"next":"finalize"}`); err != nil {
		t.Fatalf("advance critique->finalize with a skip marker should be allowed: %v", err)
	}
}

func TestStageGate_CritiqueToFinalize_BlockedByOpenBlockingFinding(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)
	ex.currentStage = StageCritique

	conv, err := st.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	lm, extra := loadLedger(conv.Meta)
	saveCriticState(extra, criticState{RoundCount: 1})
	saveCriticFindings(extra, []CriticFinding{{ID: "c1", Section: "goals", Severity: "blocking", Kind: "omission", Issue: "no measurable objective", Status: FindingOpen}})
	metaJSON, err := saveLedger(lm, extra)
	if err != nil {
		t.Fatalf("save ledger: %v", err)
	}
	if err := st.UpdateConversationMeta(ctx, ex.actorID, ex.conversationID, metaJSON); err != nil {
		t.Fatalf("update conversation meta: %v", err)
	}

	if _, err := ex.advanceStage(ctx, `{"next":"finalize"}`); err == nil {
		t.Fatal("advance with an open blocking finding should be denied, was allowed")
	}

	if _, err := ex.resolveFinding(ctx, `{"id":"c1","status":"overridden"}`); err != nil {
		t.Fatalf("resolve_finding: %v", err)
	}
	if _, err := ex.advanceStage(ctx, `{"next":"finalize"}`); err != nil {
		t.Fatalf("advance after the blocking finding was overridden should be allowed: %v", err)
	}
}
