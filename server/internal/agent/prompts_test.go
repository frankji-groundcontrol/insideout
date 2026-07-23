package agent

import (
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
