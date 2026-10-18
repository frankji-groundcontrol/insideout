package api

import "encoding/json"

// applyConversationMeta fills the parts of the conversation view that live in
// agent_conversations.meta. The key names are the agent package's (pendingStage,
// critic_state); api reads them without importing it, and
// conversations_trust_test.go pins them against stored meta.
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
	}
	if json.Unmarshal(raw, &meta) != nil {
		return
	}
	view.PendingStage = meta.PendingStage
	view.CriticSkipped = meta.CriticState.Skipped
	view.CriticRounds = meta.CriticState.RoundCount
}
