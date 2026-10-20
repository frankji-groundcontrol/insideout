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
