package agent

import (
	"strings"
	"testing"
)

// rv4-FR-F94 and F86: a terse answer ("现在就要", "ASAP", "Students.") weighs less
// than the 12-letter quote floor, and a fact written in another language than
// its quote is refused as unsupported. The model has nothing but the refusal to
// go on, so the refusal has to say how to recover: ask for a full sentence, and
// write the fact in the quote's own language. (The server side of letting a
// whole short message stand as its own quote is tested in
// rv4_review_agent_whole_test.go.)

func TestRvAgentCheckFactEvidence_ATerseQuoteRefusalSaysHowToRecover(t *testing.T) {
	for _, quote := range []string{"Students.", "现在就要", "ASAP"} {
		t.Run(quote, func(t *testing.T) {
			err := checkFactEvidence(quote, quote)
			if err == nil || !strings.Contains(err.Error(), "too short") {
				t.Fatalf("checkFactEvidence(%q, %q) = %v, want the \"too short\" refusal", quote, quote, err)
			}
			for _, want := range []string{"follow-up", "sentence"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("refusal = %q, want it to say %q: when the user's answer really is that short the model must be told to ask for more, not left to retry the same call", err, want)
				}
			}
		})
	}
}

func TestRvAgentCheckFactEvidence_ACrossLanguageRefusalNamesTheQuotesLanguage(t *testing.T) {
	cases := []struct{ name, text, quote string }{
		{"English text under a Chinese quote",
			"launch before the next cohort starts", "我们想在下一批学员开始之前上线这个功能"},
		{"Chinese text under an English quote",
			"我们想在下一批学员开始之前上线这个功能", "we want to launch before the next cohort starts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkFactEvidence(tc.text, tc.quote)
			if err == nil || !strings.Contains(err.Error(), "not supported") {
				t.Fatalf("checkFactEvidence(%q, %q) = %v, want the \"not supported\" refusal", tc.text, tc.quote, err)
			}
			if !strings.Contains(err.Error(), "language") {
				t.Fatalf("refusal = %q, want it to say the fact is written in the quote's language: a text that shares no characters with its quote can only be a translation", err)
			}
		})
	}
}
