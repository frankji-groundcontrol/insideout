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
