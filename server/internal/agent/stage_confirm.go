package agent

import (
	"context"
	"encoding/json"
)

// proposeStage parks the next stage in conversation meta. The stage column
// stays put until the person confirms.
func (e *toolExecutor) proposeStage(ctx context.Context, next string) error {
	conv, err := e.store.GetConversationForOwner(ctx, e.conversationID, e.actorID)
	if err != nil {
		return err
	}
	lm, extra := loadLedger(conv.Meta)
	b, err := json.Marshal(next)
	if err != nil {
		return err
	}
	extra["pendingStage"] = b
	metaJSON, err := saveLedger(lm, extra)
	if err != nil {
		return err
	}
	return e.store.UpdateConversationMeta(ctx, e.actorID, e.conversationID, metaJSON)
}
