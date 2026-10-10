package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

// Review-3 prompt-contract tests (cluster w3-prompts-contract): the
// clarify prompt must honor PRODUCT.md's coaching contract (AIDEV-08),
// and both findings parsers must validate section keys against the
// canonical eight (AIDEV-12).

// AIDEV-08: PRODUCT.md's coaching contract (lines 87-100) retired the
// template-filler interview — no fixed question count, no fixed
// perspective rotation, no unconditional one-question-per-turn — and
// principle 2 (lines 47-48) licenses the four-way recommendation with a
// human-decides rider.
func TestSystemPromptClarifyFollowsCoachingContract(t *testing.T) {
	prompt := systemPrompt(StageClarify, "Contract", map[string]string{}, "", "")
	for _, want := range []string{
		"相互独立的问题可以合并成一组一次提出",      // grouping allowed (PRODUCT.md:99)
		"只有当一个答案会改变访谈的下一个分支",      // one-at-a-time only when it branches (PRODUCT.md:100)
		"主动提议现在成版",                // form a version once critical questions are answered (PRODUCT.md:102)
		"建议继续深化、先去验证、暂停、或者放弃当前方向", // four-way recommendation (PRODUCT.md:47)
		"由用户决定", // the human-decides rider (PRODUCT.md:48)
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("clarify prompt missing %q — PRODUCT.md's coaching contract", want)
		}
	}
	for _, retired := range []string{
		"轮换视角",    // fixed perspective rotation
		"每个视角最多问", // per-perspective caps
		"8 个问题",   // fixed question tally
	} {
		if strings.Contains(prompt, retired) {
			t.Errorf("clarify prompt still hardcodes the retired discipline %q", retired)
		}
	}
}

// The recommendation authority is Coach-wide (PRODUCT.md principle 2), so
// every stage prompt carries it — a dying direction must be able to hear
// "consider abandoning" wherever it becomes visible.
func TestSystemPromptEveryStageCarriesRecommendationAuthority(t *testing.T) {
	const clause = "建议继续深化、先去验证、暂停、或者放弃当前方向"
	for _, stage := range stageOrder {
		if prompt := systemPrompt(stage, "R", map[string]string{}, "", ""); !strings.Contains(prompt, clause) {
			t.Errorf("%s-stage prompt must carry the four-way recommendation clause", stage)
		}
	}
}

// AIDEV-12: a ranked line whose section token is not one of the 8
// canonical PRD section keys must be dropped — a stored "roadmap" finding
// would point the round-2 critic and the panel at a section that doesn't
// exist. Drop, don't fail the batch (the established policy for bad
// severity/kind).
func TestParseRankedCritique_DropsUnknownSection(t *testing.T) {
	reply := strings.Join([]string{
		"1. [blocking] roadmap — 目标不可衡量。",
		"2. [major] goals — 目标用户写成了「所有人」。",
	}, "\n")

	got := parseRankedCritique(reply)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1 (the roadmap one dropped): %+v", len(got), got)
	}
	if got[0].Section != "goals" {
		t.Fatalf("surviving finding should be the goals one, got %+v", got[0])
	}
}

// AIDEV-12, second parser: report_findings entries with a section key
// outside store.PrdSectionKeys are dropped the same way.
func TestParseCriticFindings_DropsUnknownSection(t *testing.T) {
	args, _ := json.Marshal(map[string]any{
		"findings": []map[string]any{
			{"section": "roadmap", "severity": "blocking", "kind": "omission", "issue": "no such section"},
			{"section": "risks", "severity": "minor", "kind": "omission", "issue": "a real section"},
		},
	})
	out, err := parseCriticFindings(string(args), sectionsFixture())
	if err != nil {
		t.Fatalf("parseCriticFindings: %v", err)
	}
	if len(out) != 1 || out[0].Section != "risks" {
		t.Fatalf("unknown-section finding should be dropped, got %+v", out)
	}
}
