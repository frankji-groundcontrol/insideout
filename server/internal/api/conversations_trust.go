package api

import "encoding/json"

// pendingResolutionView is one resolve_finding request the coach parked, joined
// to the finding it names. Status is what the person's confirmation will set
// (rv4-AIDEV-03). Its JSON names are the stored finding's, so one struct reads
// both.
type pendingResolutionView struct {
	ID       string `json:"id"`
	Section  string `json:"section"`
	Severity string `json:"severity"`
	Issue    string `json:"issue"`
	Status   string `json:"status"`
}

// applyConversationMeta fills the parts of the conversation view that live in
// agent_conversations.meta. The key names are the agent package's (pendingStage,
// critic_state, critic_findings, pendingFindingResolutions); api reads them
// without importing it, and the conversations_trust tests pin them against
// stored meta.
func applyConversationMeta(view *conversationView, raw json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	var meta struct {
		PendingStage string `json:"pendingStage"`
		CriticState  struct {
			RoundCount int    `json:"roundCount"`
			Skipped    string `json:"skipped"`
		} `json:"critic_state"`
		Findings []pendingResolutionView `json:"critic_findings"`
		Parked   []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"pendingFindingResolutions"`
	}
	if json.Unmarshal(raw, &meta) != nil {
		return
	}
	view.PendingStage = meta.PendingStage
	view.CriticSkipped = meta.CriticState.Skipped
	view.CriticRounds = meta.CriticState.RoundCount

	known := make(map[string]pendingResolutionView, len(meta.Findings))
	for _, f := range meta.Findings {
		known[f.ID] = f
	}
	for _, p := range meta.Parked {
		f, ok := known[p.ID]
		// The rules ApplyStageConfirmation applies: a finding that is gone, or a
		// status other than these two, is skipped there, so it is not shown here.
		if !ok || (p.Status != "resolved" && p.Status != "overridden") {
			continue
		}
		f.Status = p.Status
		view.PendingResolutions = append(view.PendingResolutions, f)
	}
}
