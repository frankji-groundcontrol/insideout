package agent

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

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
