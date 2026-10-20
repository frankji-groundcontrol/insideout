package agent

import (
	"strings"
	"testing"
)

// record_fact must pass the conversation's user messages into
// checkFactEvidence. A four-character answer that is the whole message is
// enough to store; the same four characters lifted from a longer message
// stay too short. This goes through record_fact, not the check directly.
func TestRecordFact_WholeFourCharacterAnswerIsStored(t *testing.T) {
	const answer = "ASAP"

	t.Run("lifted from a longer message", func(t *testing.T) {
		said := "we need it ASAP, before the next cohort"
		facts, err := recordOne(t, said, "whynow", answer, answer)
		if err == nil || !strings.Contains(err.Error(), "too short") {
			t.Fatalf("record_fact of %q lifted from a longer message = %v, want it refused as too short; ledger %+v", answer, err, facts)
		}
		if len(facts) != 0 {
			t.Fatalf("a refused fact was stored: %+v", facts)
		}
	})

	t.Run("the whole message", func(t *testing.T) {
		facts, err := recordOne(t, answer, "whynow", answer, answer)
		if err != nil {
			t.Fatalf("record_fact of kind whynow using the whole four-character answer as text and quote: %v", err)
		}
		if len(facts) != 1 || facts[0].Kind != "whynow" || facts[0].Text != answer || facts[0].Quote != answer {
			t.Fatalf("ledger = %+v, want the four-character whynow fact stored", facts)
		}
	})
}
