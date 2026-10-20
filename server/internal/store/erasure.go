package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// EraseUser tombstones one account in a single transaction. Private rows
// (conversations and their messages, runs and their events, ideas of every
// status, sessions, tokens, memberships) are deleted. The users row stays:
// email, username, password hash and profile text are overwritten with
// placeholders, and the password hash does not verify. PRDs, revisions,
// commits and projects stay, still attributed to this user id. A workspace
// this user created moves to a remaining member, or is deleted when there
// is none.
func (s *Store) EraseUser(ctx context.Context, userID uuid.UUID) error {
	return s.withUserContext(ctx, userID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT id FROM insideout.workspaces WHERE creator_id = $1`, userID)
		if err != nil {
			return err
		}
		var created []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			created = append(created, id)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()

		for _, wsID := range created {
			var moved bool
			if err := tx.QueryRow(ctx, `SELECT insideout.reassign_workspace_creator($1, $2)`, wsID, userID).Scan(&moved); err != nil {
				return err
			}
			if moved {
				continue
			}
			if _, err := tx.Exec(ctx, `DELETE FROM insideout.workspaces WHERE id = $1`, wsID); err != nil {
				return err
			}
		}

		// Conversations first so a surviving run's conversation_id is set
		// null by the foreign key before the run itself is deleted.
		statements := []string{
			`DELETE FROM insideout.agent_conversations WHERE user_id = $1`,
			`DELETE FROM insideout.ai_runs WHERE user_id = $1`,
			`DELETE FROM insideout.ideas WHERE author_id = $1`,
			`DELETE FROM insideout.sessions WHERE user_id = $1`,
			`DELETE FROM insideout.auth_tokens WHERE user_id = $1`,
			`DELETE FROM insideout.workspace_memberships WHERE user_id = $1`,
		}
		for _, q := range statements {
			if _, err := tx.Exec(ctx, q, userID); err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx, `SELECT insideout.tombstone_user($1)`, userID)
		return err
	})
}
