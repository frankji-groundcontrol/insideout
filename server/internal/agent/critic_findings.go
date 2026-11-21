package agent

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// CriticFinding is one item from a critic pass, persisted in
// agent_conversations.meta.critic_findings — the critique→finalize gate
// reads this list, never the model's say-so (plan §4.4).
type CriticFinding struct {
	ID         string `json:"id"`
	Section    string `json:"section"`
	Severity   string `json:"severity"` // blocking | major | minor
	Kind       string `json:"kind"`     // defect (quotes the PRD) | omission (names what's missing)
	Quote      string `json:"quote,omitempty"`
	Issue      string `json:"issue"`
	Suggestion string `json:"suggestion,omitempty"`
	Status     string `json:"status"` // open | resolved | overridden
}

const (
	FindingOpen       = "open"
	FindingResolved   = "resolved"
	FindingOverridden = "overridden"

	maxCriticRounds = 2
)

func loadCriticFindings(extra map[string]json.RawMessage) []CriticFinding {
	var out []CriticFinding
	if v, ok := extra["critic_findings"]; ok {
		_ = json.Unmarshal(v, &out)
	}
	return out
}

func saveCriticFindings(extra map[string]json.RawMessage, findings []CriticFinding) {
	b, err := json.Marshal(findings)
	if err != nil {
		return
	}
	extra["critic_findings"] = b
}

// criticState is the small bit of round-tracking the finalize gate and
// the round cap need, stored alongside the findings under the same
// "critic_round_count" / "critic_skipped" extra keys.
type criticState struct {
	RoundCount int    `json:"roundCount"`
	Skipped    string `json:"skipped,omitempty"` // reason, e.g. "contention" — empty means not skipped
}

func loadCriticState(extra map[string]json.RawMessage) criticState {
	var cs criticState
	if v, ok := extra["critic_state"]; ok {
		_ = json.Unmarshal(v, &cs)
	}
	return cs
}

func saveCriticState(extra map[string]json.RawMessage, cs criticState) {
	b, err := json.Marshal(cs)
	if err != nil {
		return
	}
	extra["critic_state"] = b
}

// mergeRankedCritique folds a restated ranked list into the stored
// findings (AIDEV-01, review 3). The stored list is what the finalize
// gate reads, so a restatement may never shrink it: stored findings with
// no restated line survive (an omitted open blocking finding stays), a
// restated line that matches a stored finding by (section, severity,
// issue) keeps the stored entry — id and recorded status included — and
// only genuinely new lines are added, open. The issue comparison goes
// through normalizeForMatch so a restated line still matches its stored
// original when punctuation or spacing differs; a paraphrase simply
// counts as a new line, which fails safe (the original survives too).
//
// There is deliberately no "full replace on the first ranked reply"
// branch: the independent critic pass can already have stored findings
// before the coach's first ranked reply (it runs on any critique-stage
// turn, and a reply that isn't a ranked list captures nothing), and a
// replace in that window is exactly the erasure this merge exists to
// prevent. On an empty stored list, merge and replace coincide.
func mergeRankedCritique(stored, restated []CriticFinding) []CriticFinding {
	out := append([]CriticFinding(nil), stored...)
	for _, r := range restated {
		matched := false
		for i := range out {
			if out[i].Section == r.Section && out[i].Severity == r.Severity &&
				normalizeForMatch(out[i].Issue) == normalizeForMatch(r.Issue) {
				matched = true
				break
			}
		}
		if !matched {
			out = append(out, r)
		}
	}
	return out
}

// recordCriticSkip stores why this critic pass did not run, once, and tells the
// person. A skip satisfies the finalize gate on purpose (plan §5.1: a busy or
// failing critic must not hold a PRD hostage), so it must never pass unseen:
// the stream frame announces it now, and the same marker feeds the
// conversation view and the finalize confirm card (rv4-AIDEV-02).
func (c *Coach) recordCriticSkip(ctx context.Context, actorID, conversationID uuid.UUID, lm ledgerMeta, extra map[string]json.RawMessage, cs criticState, reason string, sse *sseWriter) {
	if cs.Skipped != "" {
		return // already recorded and announced for this round — don't keep re-writing
	}
	cs.Skipped = reason
	saveCriticState(extra, cs)
	metaJSON, err := saveLedger(lm, extra)
	if err != nil {
		c.log.Error("save critic skip", "error", err)
		return
	}
	if err := c.store.UpdateConversationMeta(ctx, actorID, conversationID, metaJSON); err != nil {
		c.log.Error("persist critic skip", "error", err)
		return
	}
	sse.criticSkipped(reason, cs.RoundCount)
}
