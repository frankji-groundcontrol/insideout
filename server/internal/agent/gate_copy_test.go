package agent

import (
	"strings"
	"testing"
)

// AIDEV-09: fact status "attested" is granted by the model's own
// record_fact (a 70% bigram quote match) — no human ever confirms a fact.
// The draft gate's denial copy is relayed to the user verbatim by the
// coach, so it must name the facts as recorded and never say "attested",
// which reads as a human vouching for them.
func TestStageGate_CopySaysRecordedNotAttested(t *testing.T) {
	st := newTestStore(t)
	_, ex := seedConversation(t, st)

	_, err := ex.advanceStage(t.Context(), `{"next":"draft"}`)
	if err == nil {
		t.Fatal("advance clarify->draft with zero facts should be denied, was allowed")
	}
	if strings.Contains(err.Error(), "attest") {
		t.Fatalf("gate copy implies a human attested the facts: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "recorded fact") {
		t.Fatalf("gate copy should name the missing facts as recorded: %q", err.Error())
	}
}
