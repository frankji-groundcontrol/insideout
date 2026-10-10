package agent

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// FEAT-01 (review 3): PRODUCT.md makes English and Simplified Chinese
// first-class product languages, so the conversation's locale selects the
// prompt language end to end — coach turns and critic passes alike. The
// English variants mirror the Chinese prompts' semantics exactly; zh stays
// the default for anything but an explicit 'en' (back-compat with every
// conversation created before the locale column existed).

// One unmistakable opening sentence per stage body, per language.
var stageBodyMarkers = map[string][2]string{
	StageClarify:  {`You are InsideOut's PRD coach, in the "clarify" stage`, `你是 InsideOut 的 PRD 教练，正在「澄清」阶段`},
	StageDraft:    {`You are InsideOut's PRD coach, in the "draft" stage`, `你是 InsideOut 的 PRD 教练，正在「起草」阶段`},
	StageCritique: {`You are InsideOut's PRD coach, in the "critique" stage`, `你是 InsideOut 的 PRD 教练，正在「批判」阶段`},
	StageFinalize: {`You are InsideOut's PRD coach, in the "finalize" stage`, `你是 InsideOut 的 PRD 教练，正在「定稿」阶段`},
}

func TestSystemPrompt_SelectsTheConversationsLanguage(t *testing.T) {
	for _, stage := range stageOrder {
		en := systemPrompt(localeEN, stage, "T", map[string]string{}, "", "")
		zh := systemPrompt(localeZH, stage, "T", map[string]string{}, "", "")
		markers := stageBodyMarkers[stage]
		if !strings.Contains(en, markers[0]) {
			t.Errorf("%s: en prompt lacks its English body", stage)
		}
		if strings.Contains(en, markers[1]) {
			t.Errorf("%s: en prompt still carries the Chinese body", stage)
		}
		if !strings.Contains(zh, markers[1]) {
			t.Errorf("%s: zh prompt lost its Chinese body", stage)
		}
		if strings.Contains(zh, markers[0]) {
			t.Errorf("%s: zh prompt carries the English body", stage)
		}
	}

	// Unknown and empty locales coach in zh: that is the default every
	// pre-locale conversation relies on.
	for _, unknown := range []string{"", "fr", "en_US-with-typos"} {
		if p := systemPrompt(unknown, StageClarify, "T", map[string]string{}, "", ""); !strings.Contains(p, stageBodyMarkers[StageClarify][1]) {
			t.Errorf("locale %q must default to the Chinese prompt", unknown)
		}
	}
}

// The disciplines and the fence clause are what keep an en conversation
// inside the same stage discipline and tool contracts as a zh one.
func TestSystemPromptEN_MirrorsTheDisciplinesAndFence(t *testing.T) {
	for _, stage := range stageOrder {
		p := systemPrompt(localeEN, stage, "T", map[string]string{}, "", "")
		for _, want := range []string{toolCallDisciplineEN, factDisciplineEN, recommendationDisciplineEN, dataFenceClause} {
			if !strings.Contains(p, want) {
				t.Errorf("%s: en prompt misses the mirrored discipline/fence clause starting %q", stage, want[:30])
			}
		}
		// PRODUCT.md principle 2 rides on every stage, in the chosen language.
		if strings.Contains(p, recommendationDiscipline) {
			t.Errorf("%s: en prompt carries the Chinese recommendation clause", stage)
		}
	}
}

// The critique reply format is parsed by parseRankedCritique, which is
// locale-blind: the en prompt's "Suggestion:" marker must split like the
// zh 「建议：」 one, or the panel would lose every English suggestion.
func TestParseRankedCritique_SplitsTheEnglishSuggestionMarker(t *testing.T) {
	reply := strings.Join([]string{
		"1. [blocking] goals — no measurable objective. Suggestion: add one metric per goal",
		"2. [major] users — named as everyone. Suggestion: name a specific group",
	}, "\n")

	got := parseRankedCritique(reply)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2: %+v", len(got), got)
	}
	// The trailing period stays on the issue, exactly as the zh marker
	// keeps its trailing 。 before 「建议：」.
	if got[0].Suggestion != "add one metric per goal" || got[0].Issue != "no measurable objective." {
		t.Fatalf("en suggestion not split off: issue=%q suggestion=%q", got[0].Issue, got[0].Suggestion)
	}
	if got[1].Suggestion != "name a specific group" {
		t.Fatalf("second en suggestion not split off: %+v", got[1])
	}
}

func TestRunCritic_SelectsTheConversationsLanguage(t *testing.T) {
	for _, tc := range []struct {
		locale, wantEN, wantZH string
	}{
		{localeEN, "You are a rigorous PRD reviewer", ""},
		{localeZH, "", "你是一位严格的 PRD 评审员"},
	} {
		stub := &forcingStub{}
		if _, _, ok := runCritic(context.Background(), stub, tc.locale, "T", sectionsFixture(), ledgerMeta{}, nil); !ok {
			t.Fatalf("runCritic(%s) should succeed against the stub", tc.locale)
		}
		if tc.wantEN != "" && !strings.Contains(stub.sawSystem, tc.wantEN) {
			t.Errorf("critic(%s) system prompt is not the English one", tc.locale)
		}
		if tc.wantZH != "" && !strings.Contains(stub.sawSystem, tc.wantZH) {
			t.Errorf("critic(%s) system prompt is not the Chinese one", tc.locale)
		}
	}
}

// --- end-to-end: the locale travels from conversation creation to the
// first turn's prompt assembly (DB-gated, no mocks). ---

// seedConversationInLocale is seedConversation with the locale threaded:
// the conversation is created exactly the way the convert route creates
// one for the client's chosen language.
func seedConversationInLocale(t *testing.T, st *store.Store, locale string) *toolExecutor {
	t.Helper()
	ctx := context.Background()
	u, err := st.CreateUser(ctx, "locale-"+uuid.NewString()+"@test.local", "x", "locale-tester")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, u.ID, "Locale Test WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	// Bodyless idea: the coach-turn tests own the transcript they assert.
	idea, err := st.CreateIdea(ctx, u.ID, ws.ID, "an idea", "")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	prd, conv, err := st.ConvertIdeaWithLocale(ctx, u.ID, idea.ID, locale)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}
	return &toolExecutor{store: st, actorID: u.ID, prdID: prd.ID, conversationID: conv.ID, currentStage: StageClarify}
}

// firstSystemPrompt runs one coach turn and returns the first system
// prompt the provider was handed.
func firstSystemPrompt(t *testing.T, st *store.Store, ex *toolExecutor) string {
	t.Helper()
	run := startRun(t, st, ex)
	var systems []string
	streamer := fakeStreamer{chat: func(system string, _ func(string)) (Turn, error) {
		systems = append(systems, system)
		return Turn{Text: "ok"}, nil
	}}
	c := New(st, streamer, discardLog())
	if _, err := c.runLoop(context.Background(), ex.actorID, run.ID, StageClarify, "T", map[string]string{}, nil, ex, nil); err != nil {
		t.Fatalf("runLoop: %v", err)
	}
	if len(systems) == 0 {
		t.Fatal("the turn never reached the provider")
	}
	return systems[0]
}

func TestRunLoop_PromptSpeaksTheConversationsLocale(t *testing.T) {
	st := newTestStore(t)
	wantEN, wantZH := stageBodyMarkers[StageClarify][0], stageBodyMarkers[StageClarify][1]

	if p := firstSystemPrompt(t, st, seedConversationInLocale(t, st, localeEN)); !strings.Contains(p, wantEN) || strings.Contains(p, wantZH) {
		t.Fatalf("an en conversation's first prompt is not English:\n%s", firstChars(p, 200))
	}
	if p := firstSystemPrompt(t, st, seedConversationInLocale(t, st, localeZH)); !strings.Contains(p, wantZH) || strings.Contains(p, wantEN) {
		t.Fatalf("a zh conversation's first prompt is not Chinese:\n%s", firstChars(p, 200))
	}
}

// The pre-locale ConvertIdea (zh default) keeps coaching in Chinese.
func TestRunLoop_NoLocaleConversionCoachesChinese(t *testing.T) {
	st := newTestStore(t)
	_, ex := seedConversation(t, st)

	if p := firstSystemPrompt(t, st, ex); !strings.Contains(p, stageBodyMarkers[StageClarify][1]) {
		t.Fatalf("a default conversion's first prompt is not Chinese:\n%s", firstChars(p, 200))
	}
}

func TestMaybeRunCritic_SpeaksTheConversationsLocale(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	ex := seedConversationInLocale(t, st, localeEN)
	run := startRun(t, st, ex)

	// Park a round-1 pass so the critic runs and re-reviews open findings.
	extra := map[string]json.RawMessage{}
	saveCriticFindings(extra, []CriticFinding{{
		ID: "c1", Section: "goals", Severity: "major", Kind: "omission", Issue: "x", Status: FindingOpen,
	}})
	saveCriticState(extra, criticState{RoundCount: 1})
	meta, err := saveLedger(ledgerMeta{}, extra)
	if err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if err := st.UpdateConversationMeta(ctx, ex.actorID, ex.conversationID, meta); err != nil {
		t.Fatalf("store meta: %v", err)
	}

	stub := &forcingStub{}
	c := New(st, stub, discardLog())
	sse, ok := newSSEWriter(httptest.NewRecorder())
	if !ok {
		t.Fatal("recorder cannot flush")
	}
	c.maybeRunCritic(ctx, ex.actorID, ex.prdID, ex.conversationID, run.ID, StageCritique, "no ranked list", true, false, sse)

	if !strings.Contains(stub.sawSystem, "You are a rigorous PRD reviewer") {
		t.Fatalf("an en conversation's critic pass is not English:\n%s", firstChars(stub.sawSystem, 200))
	}
}

func firstChars(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// The planner drafts in the conversation's language too: the en prompts
// mirror the zh ones (roadmap_planner.go), fence clause included.
func TestPlannerPrompts_SelectTheConversationsLanguage(t *testing.T) {
	const enMarker, zhMarker = "You are InsideOut's product-delivery coach", "你是 InsideOut 的产品落地教练"

	stub := &forcingStub{}
	_, _ = NewAnthropicRoadmapPlanner(stub).PlanMVP(context.Background(), localeEN, "T", sectionsFixture())
	if !strings.Contains(stub.sawSystem, enMarker) || strings.Contains(stub.sawSystem, zhMarker) {
		t.Fatalf("en PlanMVP system prompt is not the English one:\n%s", firstChars(stub.sawSystem, 200))
	}
	if !strings.Contains(stub.sawSystem, dataFenceClause) {
		t.Error("en PlanMVP prompt lost the language-independent fence clause")
	}

	stub = &forcingStub{}
	_, _ = NewAnthropicRoadmapPlanner(stub).PlanMVP(context.Background(), localeZH, "T", sectionsFixture())
	if !strings.Contains(stub.sawSystem, zhMarker) {
		t.Fatal("zh PlanMVP system prompt lost the Chinese body")
	}

	stub = &forcingStub{}
	_, _ = NewAnthropicRoadmapPlanner(stub).ExpandNode(context.Background(), localeEN, "P", "N", "D")
	if !strings.Contains(stub.sawSystem, enMarker) || strings.Contains(stub.sawSystem, zhMarker) {
		t.Fatalf("en ExpandNode system prompt is not the English one:\n%s", firstChars(stub.sawSystem, 200))
	}

	stub = &forcingStub{}
	_, _ = NewAnthropicRoadmapPlanner(stub).ExpandNode(context.Background(), localeZH, "P", "N", "D")
	if !strings.Contains(stub.sawSystem, zhMarker) {
		t.Fatal("zh ExpandNode system prompt lost the Chinese body")
	}
}
