package agent

import (
	"context"
	"encoding/json"
	"fmt"
)

func (e *toolExecutor) resolveFinding(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("agent: invalid resolve_finding arguments: %w", err)
	}
	if args.Status != FindingResolved && args.Status != FindingOverridden {
		return "", fmt.Errorf("agent: status must be %q or %q", FindingResolved, FindingOverridden)
	}

	conv, err := e.store.GetConversationForOwner(ctx, e.conversationID, e.actorID)
	if err != nil {
		return "", err
	}
	lm, extra := loadLedger(conv.Meta)
	findings := loadCriticFindings(extra)
	found := false
	for i := range findings {
		if findings[i].ID == args.ID {
			findings[i].Status = args.Status
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("agent: unknown finding id %q", args.ID)
	}
	saveCriticFindings(extra, findings)
	metaJSON, err := saveLedger(lm, extra)
	if err != nil {
		return "", err
	}
	if err := e.store.UpdateConversationMeta(ctx, e.actorID, e.conversationID, metaJSON); err != nil {
		return "", err
	}
	return fmt.Sprintf("finding %s marked %s", args.ID, args.Status), nil
}
