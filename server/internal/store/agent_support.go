package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PrdByProject returns the PRD linked to a project (there is at most
// one), for agent context assembly. RLS applies with actorID.
func (s *Store) PrdByProject(ctx context.Context, actorID, projectID uuid.UUID) (*Prd, error) {
	var p Prd
	var ideaID, projID *uuid.UUID
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT `+prdColumns+` FROM insideout.prds
			WHERE project_id = $1 ORDER BY created_at LIMIT 1`, projectID,
		).Scan(&p.ID, &p.WorkspaceID, &ideaID, &projID, &p.AuthorID, &p.Title, &p.Sections, &p.Status, &p.CurrentRevision, &p.CreatedAt, &p.UpdatedAt)
	})
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.IdeaID, p.ProjectID = ideaID, projID
	return &p, nil
}

// EvidenceCountsByProject counts evidence rows per roadmap node of a
// project, for the compact agent context.
func (s *Store) EvidenceCountsByProject(ctx context.Context, actorID, projectID uuid.UUID) (map[string]int, error) {
	out := map[string]int{}
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT e.node_id::text, count(*)
			FROM insideout.roadmap_evidence e
			JOIN insideout.roadmap_nodes n ON n.id = e.node_id
			WHERE n.project_id = $1
			GROUP BY e.node_id`, projectID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			var n int
			if err := rows.Scan(&id, &n); err != nil {
				return err
			}
			out[id] = n
		}
		return rows.Err()
	})
	return out, err
}
