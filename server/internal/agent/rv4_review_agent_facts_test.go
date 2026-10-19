package agent

import (
	"strings"
	"testing"
)

// rv4-FR-F90: the support check measured only the share of the text that its
// quote holds, so a verbatim quote followed by a much longer invented claim
// cleared it (support 0.27 against a floor of 0.2) and was stored as an attested
// fact that later prompts and the stage gates take as ground truth. A text that
// outgrows its quote has to be built mostly from it.

const (
	rvAgentLaunchQuote = "we want to launch before the next cohort starts"
	rvAgentPadding     = ". The user also approved a 40 percent price increase for all enterprise customers and agreed to waive the security review entirely"
)

func TestRvAgentCheckFactEvidence_AQuoteWithAnInventedTailIsRefused(t *testing.T) {
	cases := []struct{ name, text, quote string }{
		{"the review's probe", rvAgentLaunchQuote + rvAgentPadding, rvAgentLaunchQuote},
		{"a shorter invented tail",
			"our designers lose old notes every week and enterprise buyers also need SOC 2 compliance",
			"our designers lose old notes every week"},
		{"an invented tail on a long quote",
			"nobody can find their old notes once the workshop moves on and enterprise customers will pay forty dollars per seat for dark mode",
			"nobody can find their old notes once the workshop moves on"},
		{"Chinese",
			"设计师每周丢失笔记，而且大型企业客户愿意为深色模式和离线同步每个席位支付四十美元",
			"设计师每周丢失笔记"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkFactEvidence(tc.text, tc.quote)
			if err == nil || !strings.Contains(err.Error(), "not supported") {
				t.Fatalf("checkFactEvidence(%q, %q) = %v, want a \"not supported\" refusal: most of the text is not in the quote", tc.text, tc.quote, err)
			}
		})
	}
}

// Through the real record_fact, against PostgreSQL: the padded claim must not
// reach the ledger as an attested decision.
func TestRvAgentRecordFact_APaddedDecisionIsNotStored(t *testing.T) {
	facts, err := recordOne(t, rvAgentLaunchQuote, "decision", rvAgentLaunchQuote+rvAgentPadding, rvAgentLaunchQuote)
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("record_fact of a quote plus an invented price decision = %v, want a \"not supported\" refusal", err)
	}
	if len(facts) != 0 {
		t.Fatalf("the ledger holds %+v: an invented decision was stored as attested", facts)
	}
}

// What the bound must leave alone: a longer text that is still the quote's own
// words (a pronoun swap), and the quote itself.
func TestRvAgentCheckFactEvidence_ALongerTextMostlyFromItsQuoteStillPasses(t *testing.T) {
	cases := []struct{ name, text, quote string }{
		{"a pronoun swap", "they want it soon", "we want it soon"},
		{"the quote verbatim", rvAgentLaunchQuote, rvAgentLaunchQuote},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := checkFactEvidence(tc.text, tc.quote); err != nil {
				t.Fatalf("checkFactEvidence(%q, %q) = %v, want accepted", tc.text, tc.quote, err)
			}
		})
	}
}
