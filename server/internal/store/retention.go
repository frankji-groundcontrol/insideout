package store

import (
	"context"
	"time"
)

// conversationRetention is how long a coach conversation lives after its
// newest message. A conversation with no messages is dated from its own
// creation. Each new message resets the clock. The PRD is a different row
// and is not purged. 365 days is the window the owner set (rv4-DATA-06).
const conversationRetention = 365 * 24 * time.Hour

// purgeIdleConversations deletes conversations past conversationRetention.
// Messages cascade. ai_runs.conversation_id is ON DELETE SET NULL, so a run
// inside the 30-day run window keeps its prompt with no conversation. The
// reaper calls this with no app.user_id; the conversation and message
// policies admit that system context (20261020140000).
func (s *Store) purgeIdleConversations(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, `
		DELETE FROM insideout.agent_conversations AS c
		WHERE COALESCE(
			(SELECT max(m.created_at) FROM insideout.agent_messages AS m WHERE m.conversation_id = c.id),
			c.created_at
		) < now() - make_interval(secs => $1)`,
		conversationRetention.Seconds(),
	)
	return err
}

// retentionWindow is how long rows stay queryable before the reaper's
// purge deletes them: ai_runs (carrying raw prompt/response text, and
// cascading their ai_run_events telemetry) created before the window, and
// sessions, auth_tokens, invitations and reset_watches that expired before
// it. Coach conversations use conversationRetention (above).
// Operator note: the purge is permanent and archives nothing first; raise
// this constant to keep data longer.
const retentionWindow = 30 * 24 * time.Hour

// purgePastRetention deletes what the stale flip leaves behind: old
// ai_runs (their ON DELETE CASCADE clears ai_run_events) and auth rows
// that expired long ago — the live counterparts (recent runs, unexpired
// sessions/tokens/invitations/reset watches) never match the predicates.
func (s *Store) purgePastRetention(ctx context.Context) error {
	purges := []string{
		`DELETE FROM insideout.ai_runs WHERE created_at < now() - make_interval(secs => $1)`,
		`DELETE FROM insideout.sessions WHERE expires_at < now() - make_interval(secs => $1)`,
		`DELETE FROM insideout.auth_tokens WHERE expires_at < now() - make_interval(secs => $1)`,
		`DELETE FROM insideout.invitations WHERE expires_at < now() - make_interval(secs => $1)`,
		`DELETE FROM insideout.reset_watches WHERE expires_at < now() - make_interval(secs => $1)`,
	}
	for _, q := range purges {
		if _, err := s.Pool.Exec(ctx, q, retentionWindow.Seconds()); err != nil {
			return err
		}
	}
	return nil
}
