package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// AIDEV-10: fact/assumption text and the assumption count are model-authored
// output that gets persisted in conversation meta and re-injected into every
// later system prompt, so they need the same bounds the planner puts on its
// own output — the quote is already capped (maxQuoteLen), these were not.

func oversizedText(n int) string {
	return strings.Repeat("x", n)
}

// A fact whose text exceeds the cap must be stored truncated to the cap with
// a marker, not whole.
func TestRecordFact_TruncatesOversizedText(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)
	ex.userMessages = []string{"we lose customers every week to this"}

	args, _ := json.Marshal(map[string]string{
		"kind":  "problem",
		"text":  "we lose customers every week " + oversizedText(maxTextLen+50),
		"quote": "we lose customers every week",
	})
	if _, err := ex.recordFact(ctx, string(args)); err != nil {
		t.Fatalf("record_fact with oversized text: %v", err)
	}
	lm, err := currentLedger(ctx, st, ex.actorID, ex.conversationID)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	if len(lm.Facts) != 1 {
		t.Fatalf("facts = %+v, want exactly one", lm.Facts)
	}
	stored := lm.Facts[0].Text
	if got := len([]rune(stored)); got != maxTextLen {
		t.Fatalf("stored fact text = %d runes, want exactly the %d-rune cap", got, maxTextLen)
	}
	if !strings.HasSuffix(stored, string(truncationMarker)) {
		t.Fatalf("stored fact text must end with the truncation marker %q, got tail %q", string(truncationMarker), stored[len(stored)-4:])
	}
}

// An assumption whose text exceeds the cap must be stored truncated with a
// marker, not whole.
func TestMarkAssumption_TruncatesOversizedText(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)

	args, _ := json.Marshal(map[string]string{"text": oversizedText(maxTextLen + 50)})
	if _, err := ex.markAssumption(ctx, string(args)); err != nil {
		t.Fatalf("mark_assumption with oversized text: %v", err)
	}
	lm, err := currentLedger(ctx, st, ex.actorID, ex.conversationID)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	if len(lm.Facts) != 1 {
		t.Fatalf("facts = %+v, want exactly one assumption", lm.Facts)
	}
	stored := lm.Facts[0].Text
	if got := len([]rune(stored)); got != maxTextLen {
		t.Fatalf("stored assumption text = %d runes, want exactly the %d-rune cap", got, maxTextLen)
	}
	if !strings.HasSuffix(stored, string(truncationMarker)) {
		t.Fatalf("stored assumption text must end with the truncation marker %q, got tail %q", string(truncationMarker), stored[len(stored)-4:])
	}
}

// The (maxAssumptions+1)'th mark_assumption call must be refused and not
// stored — mirroring record_fact's maxFactsPerKind rejection.
func TestMarkAssumption_RejectsBeyondCapPerConversation(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)

	mark := func(text string) error {
		args, _ := json.Marshal(map[string]string{"text": text})
		_, err := ex.markAssumption(ctx, string(args))
		return err
	}
	first := "the workshop cohort renews annually"
	if err := mark(first); err != nil {
		t.Fatalf("first mark_assumption: %v", err)
	}
	for i := 1; i < maxAssumptions; i++ {
		if err := mark(oversizedText(8)); err != nil {
			t.Fatalf("mark_assumption %d within the cap: %v", i, err)
		}
	}
	err := mark("one assumption too many")
	if err == nil || !strings.Contains(err.Error(), "assumption") {
		t.Fatalf("mark_assumption beyond the cap = %v, want a refusal naming assumptions", err)
	}
	lm, rerr := currentLedger(ctx, st, ex.actorID, ex.conversationID)
	if rerr != nil {
		t.Fatalf("read ledger: %v", rerr)
	}
	if n := countByKind(lm.Facts, "assumption"); n != maxAssumptions {
		t.Fatalf("stored assumptions = %d, want the cap of %d with the extra one rejected", n, maxAssumptions)
	}
	if lm.Facts[0].Text != first {
		t.Fatalf("in-cap assumption must be stored verbatim, got %q", lm.Facts[0].Text)
	}
}
