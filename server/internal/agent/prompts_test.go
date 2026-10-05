package agent

import (
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"strings"
	"testing"
)

func TestNextStage_LegalTransitionOrder(t *testing.T) {
	cases := []struct{ from, want string }{
		{StageClarify, StageDraft},
		{StageDraft, StageCritique},
		{StageCritique, StageFinalize},
		{StageFinalize, ""},
		{"not-a-stage", ""},
	}
	for _, c := range cases {
		if got := nextStage(c.from); got != c.want {
			t.Errorf("nextStage(%q) = %q, want %q", c.from, got, c.want)
		}
	}
}

func TestSystemPrompt_IncludesStageAndPrdContext(t *testing.T) {
	prompt := systemPrompt(StageDraft, "My PRD", map[string]string{"background": "some content"}, "- [problem/attested] users can't find X（原话：找不到）", "")
	if prompt == "" {
		t.Fatal("systemPrompt returned empty string")
	}
	if !strings.Contains(prompt, "My PRD") {
		t.Errorf("systemPrompt should mention the PRD title, got: %s", prompt)
	}
	if !strings.Contains(prompt, "some content") {
		t.Errorf("systemPrompt should include current section content, got: %s", prompt)
	}
	if !strings.Contains(prompt, "users can't find X") {
		t.Errorf("systemPrompt should include the fact ledger, got: %s", prompt)
	}
}

func TestSystemPromptWeavesReadinessGaps(t *testing.T) {
	prompt := systemPrompt(StageClarify, "Gaps", map[string]string{}, "", "")
	for _, want := range []string{
		"当前读者缺口",
		"must_clarify_now",
		"should_clarify_this_version",
		"现在成版",
		"优先级（必须现在澄清 / 本版应澄清 / 之后再验证）",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("missing %q in prompt", want)
		}
	}
}

func TestSystemPromptCompletePrDHasNoBlockingGaps(t *testing.T) {
	full := map[string]string{}
	for _, k := range store.PrdSectionKeys {
		full[k] = "written"
	}
	prompt := systemPrompt(StageDraft, "Full", full, "", "")
	if !strings.Contains(prompt, "无阻塞性缺口") {
		t.Errorf("complete PRD should show no blocking gaps; got:\n%.300s", prompt)
	}
	if strings.Contains(prompt, "must_clarify_now") {
		t.Error("complete PRD must not carry must-clarify gaps")
	}
}

func TestSystemPromptCritiqueAsksForRankedList(t *testing.T) {
	prompt := systemPrompt(StageCritique, "Ranked", map[string]string{}, "", "")
	for _, want := range []string{"按严重程度从高到低", "[blocking]", "编号列表"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("critique prompt should ask for a ranked list, missing %q", want)
		}
	}
	if strings.Contains(prompt, "一次只讲一条") {
		t.Error("critique prompt still asks the coach to narrate one finding at a time")
	}
}

// A three-item reply, most severe first, becomes the findings the panel
// renders — order preserved, severity on each item.
func TestParseRankedCritique_ThreeItems(t *testing.T) {
	reply := strings.Join([]string{
		"1. [blocking] goals — 目标不可衡量。建议：补一个可量化的指标。",
		"2. [major] users — 目标用户写成了「所有人」。建议：具体到一类人。",
		"3. [minor] risks — 没写失败时怎么办。",
	}, "\n")

	got := parseRankedCritique(reply)
	if len(got) != 3 {
		t.Fatalf("got %d findings, want 3: %+v", len(got), got)
	}
	want := []struct{ severity, section, issue string }{
		{"blocking", "goals", "目标不可衡量。"},
		{"major", "users", "目标用户写成了「所有人」。"},
		{"minor", "risks", "没写失败时怎么办。"},
	}
	for i, w := range want {
		if got[i].Severity != w.severity || got[i].Section != w.section || got[i].Issue != w.issue {
			t.Errorf("item %d = %+v, want severity %q section %q issue %q", i, got[i], w.severity, w.section, w.issue)
		}
		if got[i].Status != FindingOpen || got[i].ID == "" {
			t.Errorf("item %d should be an open finding with an id, got %+v", i, got[i])
		}
	}
	if got[0].Suggestion != "补一个可量化的指标。" || got[1].Suggestion != "具体到一类人。" {
		t.Errorf("suggestions not parsed: %+v", got)
	}
	if got[2].Suggestion != "" {
		t.Errorf("item without a suggestion should leave it empty, got %q", got[2].Suggestion)
	}
}

// The live critique turn hands its reply here. A three-item reply becomes
// the findings the panel renders; the same reply in any other stage does not.
func TestRankedCritiqueFindings_ThreeItemReply(t *testing.T) {
	reply := strings.Join([]string{
		"1. [blocking] goals — 目标不可衡量。建议：补一个可量化的指标。",
		"2. [major] users — 目标用户写成了「所有人」。建议：具体到一类人。",
		"3. [minor] risks — 没写失败时怎么办。",
	}, "\n")

	got := rankedCritiqueFindings(StageCritique, reply)
	if len(got) != 3 {
		t.Fatalf("critique reply became %d findings, want 3", len(got))
	}
	if got[0].Issue != "目标不可衡量。" || got[2].Section != "risks" {
		t.Fatalf("findings = %+v", got)
	}
	if rankedCritiqueFindings(StageDraft, reply) != nil {
		t.Fatal("a draft-stage reply must not become findings")
	}
	if rankedCritiqueFindings(StageCritique, "先看第一条。") != nil {
		t.Fatal("a reply that isn't the ranked list must not replace the findings")
	}
}
