package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PrdSummary is one row of a workspace's PRD list: enough to find a PRD and
// see where it stands, without its sections.
type PrdSummary struct {
	ID         uuid.UUID
	Title      string
	Status     string
	AuthorID   uuid.UUID
	AuthorName string
	ProjectID  *uuid.UUID
	UpdatedAt  time.Time
}

// ListWorkspacePrds returns every PRD of the workspace, newest first, to a
// member: the reviewer who must approve a PRD, or a teammate, finds it here
// rather than through its author's private idea list (rv5-FE-02, PARB-03).
// A non-member gets ErrNotFound, re-checked in the read's transaction, so the
// list does not say whether the workspace exists.
func (s *Store) ListWorkspacePrds(ctx context.Context, actorID, workspaceID uuid.UUID) ([]PrdSummary, error) {
	out := []PrdSummary{}
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if _, err := requireMember(ctx, tx, workspaceID, actorID); errors.Is(err, ErrForbidden) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			SELECT p.id, p.title, p.status, p.author_id, COALESCE(u.username, ''), p.project_id, p.updated_at
			FROM insideout.prds p
			LEFT JOIN insideout.users u ON u.id = p.author_id
			WHERE p.workspace_id = $1
			ORDER BY p.updated_at DESC, p.id`, workspaceID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var p PrdSummary
			if err := rows.Scan(&p.ID, &p.Title, &p.Status, &p.AuthorID, &p.AuthorName, &p.ProjectID, &p.UpdatedAt); err != nil {
				return err
			}
			out = append(out, p)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
