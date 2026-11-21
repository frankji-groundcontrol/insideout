package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// rv4-FR-F94 and F86, the check itself: a short answer that is the user's entire
// message may attest a fact (checkFactEvidence's userMessages). record_fact does
// not pass the messages yet, because its call in ledger.go is outside this
// cluster's files, so until it does these exercise the check directly.

func TestRvAgentCheckFactEvidence_AWholeShortAnswerAttests(t *testing.T) {
	cases := []struct {
		name, text, quote string
		said              []string
	}{
		{"four Han characters", "现在就要", "现在就要", []string{"我们的设计师每周丢失笔记", "现在就要"}},
		{"an acronym", "ASAP", "ASAP", []string{"ASAP"}},
		{"a word and its period", "Students", "Students.", []string{"who is it for?", "Students."}},
		{"case and punctuation differ", "right now", "right now", []string{"Right now!"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := checkFactEvidence(tc.text, tc.quote, tc.said...); err != nil {
				t.Fatalf("checkFactEvidence(%q, %q, %q) = %v, want the whole short answer to attest", tc.text, tc.quote, tc.said, err)
			}
		})
	}
}

// The exemption is for the whole message only: the same words lifted out of a
// longer message are the fragment the floor was written to stop.
func TestRvAgentCheckFactEvidence_AFragmentOfALongerMessageStillNeedsTheFloor(t *testing.T) {
	cases := []struct {
		name, quote string
		said        []string
	}{
		{"an acronym inside a sentence", "ASAP", []string{"we need it ASAP, before the next cohort"}},
		{"four Han characters inside a sentence", "现在就要", []string{"我们现在就要上线这个功能"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkFactEvidence(tc.quote, tc.quote, tc.said...)
			if err == nil || !strings.Contains(err.Error(), "too short") {
				t.Fatalf("checkFactEvidence(%q, %q, %q) = %v, want the \"too short\" refusal", tc.quote, tc.quote, tc.said, err)
			}
		})
	}
}

func TestRvAgentCheckFactEvidence_AWholeAnswerBelowTheMinimumIsRefused(t *testing.T) {
	for _, word := range []string{"ok", "yes", "是"} {
		t.Run(word, func(t *testing.T) {
			err := checkFactEvidence(word, word, word)
			if err == nil || !strings.Contains(err.Error(), "too short") {
				t.Fatalf("checkFactEvidence(%q, %q, %q) = %v, want the \"too short\" refusal: below %d letters even a whole message says too little", word, word, word, err, minWholeAnswerRunes)
			}
		})
	}
}

// A whole short answer clears only the length floor: its text still has to be
// about it, and a text that outgrows it still has to be mostly its words.
func TestRvAgentCheckFactEvidence_AWholeAnswerStillHasToSupportTheText(t *testing.T) {
	cases := []struct{ name, text, quote string }{
		{"text about something else", "customers want dark mode", "ASAP"},
		{"text that outgrows the answer", "我们现在就要立刻上线这个功能", "现在就要"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkFactEvidence(tc.text, tc.quote, tc.quote)
			if err == nil || !strings.Contains(err.Error(), "not supported") {
				t.Fatalf("checkFactEvidence(%q, %q, whole message) = %v, want the \"not supported\" refusal", tc.text, tc.quote, err)
			}
		})
	}
}

// The gate hint used to tell the model to "record a decision fact saying so" to
// skip a missing kind. checkStageGate counts only problem, segment, alternative
// and whynow, so a decision fact does not stand in for any of them: with one
// recorded the gate still names all four, and the model needs a hint that points
// at a fact of the missing kind instead.
func TestRvAgentCheckStageGate_ADecisionFactDoesNotStandInForAMissingKind(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)
	said := "honestly we would rather skip all of those questions and just get a draft"
	ex.userMessages = []string{said}
	args, err := json.Marshal(map[string]string{"kind": "decision", "text": "skip the questions and get a draft", "quote": said})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := ex.recordFact(ctx, string(args)); err != nil {
		t.Fatalf("record_fact decision: %v", err)
	}
	err = ex.checkStageGate(ctx, StageDraft)
	if err == nil || !strings.Contains(err.Error(), "problem, segment, alternative, whynow") {
		t.Fatalf("checkStageGate(draft) with only a decision fact = %v, want it to still name problem, segment, alternative, whynow", err)
	}
}
