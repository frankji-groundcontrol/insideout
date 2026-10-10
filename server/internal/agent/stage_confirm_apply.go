package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ApplyStageConfirmation rewrites conversation meta at the moment the person
// confirms a stage change. The finding resolutions the model parked with
// resolve_finding become the stored statuses — overrides are human-authored
// by construction, the parked requests are consumed, and a decision-trail
// entry records what was ratified and when (AIDEV-07). The pendingStage key
// is consumed here too; the stage column itself is the store's job.
func ApplyStageConfirmation(raw json.RawMessage) (json.RawMessage, error) {
	lm, extra := loadLedger(raw)
	delete(extra, "pendingStage")
	pending := loadPendingFindingResolutions(extra)
	delete(extra, "pendingFindingResolutions")
	if len(pending) > 0 {
		findings := loadCriticFindings(extra)
		byID := make(map[string]int, len(findings))
		for i, f := range findings {
			byID[f.ID] = i
		}
		applied := loadResolutionTrail(extra)
		for _, p := range pending {
			i, ok := byID[p.ID]
			if !ok {
				continue // the finding was dropped by a later restatement; nothing to ratify
			}
			if p.Status != FindingResolved && p.Status != FindingOverridden {
				continue // parked rows were validated at park time; do not honor corrupted ones
			}
			findings[i].Status = p.Status
			applied = append(applied, resolutionTrailEntry{ID: p.ID, Status: p.Status, At: time.Now().UTC().Format(time.RFC3339)})
		}
		saveCriticFindings(extra, findings)
		if len(applied) > 0 {
			b, err := json.Marshal(applied)
			if err != nil {
				return nil, err
			}
			extra["findingResolutionsApplied"] = b
		}
	}
	return saveLedger(lm, extra)
}

type resolutionTrailEntry struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	At     string `json:"at"`
}

func loadResolutionTrail(extra map[string]json.RawMessage) []resolutionTrailEntry {
	var out []resolutionTrailEntry
	if v, ok := extra["findingResolutionsApplied"]; ok {
		_ = json.Unmarshal(v, &out)
	}
	return out
}

// ConfirmStage implements the api.Coach seam: the human confirmed the stage
// the coach proposed, so the same transaction also applies the finding
// resolutions the model parked for ratification.
func (c *Coach) ConfirmStage(ctx context.Context, actorID, conversationID uuid.UUID, stage string) error {
	return c.store.ConfirmConversationStage(ctx, actorID, conversationID, stage, ApplyStageConfirmation)
}
