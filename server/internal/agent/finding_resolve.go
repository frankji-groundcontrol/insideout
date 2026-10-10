package agent

import (
	"context"
	"encoding/json"
	"fmt"
)

// pendingFindingResolution is one resolve_finding request the model
// parked in agent_conversations.meta, waiting for the human who confirms
// the stage change to ratify it (AIDEV-07): overrides must be
// human-authored by construction, so the model may propose a status but
// never apply one.
type pendingFindingResolution struct {
	ID     string `json:"id"`
	Status string `json:"status"` // resolved | overridden
}

func loadPendingFindingResolutions(extra map[string]json.RawMessage) []pendingFindingResolution {
	var out []pendingFindingResolution
	if v, ok := extra["pendingFindingResolutions"]; ok {
		_ = json.Unmarshal(v, &out)
	}
	return out
}

func savePendingFindingResolutions(extra map[string]json.RawMessage, pending []pendingFindingResolution) {
	b, err := json.Marshal(pending)
	if err != nil {
		return
	}
	extra["pendingFindingResolutions"] = b
}

// hasUnresolvedBlockingFindings is the finalize gate's view of the
// findings: an open blocking finding blocks the proposal unless a
// resolution for it is parked — the park is what the person ratifies in
// the same confirmation that advances the stage, while the stored status
// itself stays open until that confirmation applies it.
func hasUnresolvedBlockingFindings(findings []CriticFinding, pending []pendingFindingResolution) bool {
	parked := make(map[string]bool, len(pending))
	for _, p := range pending {
		parked[p.ID] = true
	}
	for _, f := range findings {
		if f.Status == FindingOpen && f.Severity == "blocking" && !parked[f.ID] {
			return true
		}
	}
	return false
}

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
	for _, f := range findings {
		if f.ID == args.ID {
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("agent: unknown finding id %q", args.ID)
	}
	// Park the request instead of applying it (AIDEV-07): the stored
	// status only changes when the person confirms, through the same
	// pending-confirm policy advance_stage uses for the stage itself.
	pending := loadPendingFindingResolutions(extra)
	replaced := false
	for i := range pending {
		if pending[i].ID == args.ID {
			pending[i].Status = args.Status
			replaced = true
			break
		}
	}
	if !replaced {
		pending = append(pending, pendingFindingResolution{ID: args.ID, Status: args.Status})
	}
	savePendingFindingResolutions(extra, pending)
	metaJSON, err := saveLedger(lm, extra)
	if err != nil {
		return "", err
	}
	if err := e.store.UpdateConversationMeta(ctx, e.actorID, e.conversationID, metaJSON); err != nil {
		return "", err
	}
	return fmt.Sprintf("finding %s marked %s — waiting for the user's confirmation", args.ID, args.Status), nil
}
