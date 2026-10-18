package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// rv4-AIDEV-07: record_fact stores a fact as "attested" as soon as its quote is
// grounded in something the user said. Only the quote was ever checked: no
// minimum length, and the fact's own text was never compared with it, so a
// four-character quote that any message contains stood behind whatever claim the
// model chose to write. No human confirms a fact, so the server's check is the
// only one there is.

// recordOne records one fact against the given user turns and returns what the
// ledger holds afterwards with the error record_fact answered.
func recordOne(t *testing.T, said, kind, text, quote string) (facts []Fact, err error) {
	t.Helper()
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)
	ex.userMessages = []string{said}
	args, mErr := json.Marshal(map[string]string{"kind": kind, "text": text, "quote": quote})
	if mErr != nil {
		t.Fatalf("marshal: %v", mErr)
	}
	_, err = ex.recordFact(ctx, string(args))
	lm, lErr := currentLedger(ctx, st, ex.actorID, ex.conversationID)
	if lErr != nil {
		t.Fatalf("read ledger: %v", lErr)
	}
	return lm.Facts, err
}

const (
	saidEN = "honestly nobody can find their old notes once the workshop moves on"
	saidZH = "我们的设计师团队每周都会丢失大量会议笔记，找不回来"
)

func TestTrustFacts_RefusesWhatTheQuoteCannotAttest(t *testing.T) {
	cases := []struct{ name, said, text, quote, want string }{
		{"a four-character quote any message holds", saidEN,
			"the product needs dark mode and offline sync", "find", "too short"},
		{"a grounded long quote under text about something else", saidEN,
			"customers want dark mode and offline sync across their devices",
			"nobody can find their old notes once the workshop moves on", "not supported"},
		{"a four-character Chinese quote", saidZH,
			"客户想要深色模式和离线同步", "会议笔记", "too short"},
		{"a grounded long Chinese quote under unrelated text", saidZH,
			"客户想要深色模式和离线同步功能", "我们的设计师团队每周都会丢失大量会议笔记", "not supported"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facts, err := recordOne(t, tc.said, "problem", tc.text, tc.quote)
			if err == nil {
				t.Fatalf("record_fact(text=%q, quote=%q) was accepted as attested; the ledger holds %+v", tc.text, tc.quote, facts)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("refusal = %q, want it to say %q so the model can fix the call", err, tc.want)
			}
			if len(facts) != 0 {
				t.Fatalf("a refused fact was stored anyway: %+v", facts)
			}
		})
	}
}

// What the check must keep letting through: a short restatement of a long quote,
// in either language, and a short Chinese quote that is a whole clause.
func TestTrustFacts_StillAcceptsAHonestRestatement(t *testing.T) {
	cases := []struct{ name, said, text, quote string }{
		{"English paraphrase", saidEN,
			"users lose track of old notes", "nobody can find their old notes once the workshop moves on"},
		{"terse Chinese restatement", saidZH,
			"笔记丢失", saidZH},
		{"a whole clause in few Chinese characters", "独立设计师每周都要重做提案",
			"独立设计师", "独立设计师每周都要重做提案"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facts, err := recordOne(t, tc.said, "problem", tc.text, tc.quote)
			if err != nil {
				t.Fatalf("record_fact(text=%q): %v", tc.text, err)
			}
			if len(facts) != 1 || facts[0].Text != tc.text {
				t.Fatalf("ledger = %+v, want the one recorded fact", facts)
			}
		})
	}
}
