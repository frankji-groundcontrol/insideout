package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ProjectUpdate struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	AuthorID  uuid.UUID
	Kind      string
	Content   string
	CreatedAt time.Time
	// Decision is set only for an agent_proposal that a human has
	// accepted or rejected. Empty means still open.
	Decision string
}

// AddProjectUpdate requires actorID to currently be a member of the
// project's workspace. A proposal changes the plan, so only the
// project owner or a workspace admin may file one; notes, progress,
// and blockers stay open to any member (BUG-020).
func (s *Store) AddProjectUpdate(ctx context.Context, actorID, projectID uuid.UUID, kind, content string) (*ProjectUpdate, error) {
	var u ProjectUpdate
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var workspaceID uuid.UUID
		err := tx.QueryRow(ctx, `SELECT workspace_id FROM insideout.projects WHERE id = $1`, projectID).Scan(&workspaceID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if _, err := requireMember(ctx, tx, workspaceID, actorID); err != nil {
			return err
		}
		if kind == "agent_proposal" {
			if err := requirePlanWriter(ctx, tx, projectID, actorID); err != nil {
				return err
			}
		}

		return tx.QueryRow(ctx, `
			INSERT INTO insideout.project_updates (project_id, author_id, kind, content)
			VALUES ($1, $2, $3, $4)
			RETURNING id, project_id, author_id, kind, content, created_at`,
			projectID, actorID, kind, content,
		).Scan(&u.ID, &u.ProjectID, &u.AuthorID, &u.Kind, &u.Content, &u.CreatedAt)
	})
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ProjectUpdatesPageSize bounds one timeline page — a long-lived project's
// history must not ship in full on every project fetch (the GetProject embed).
const ProjectUpdatesPageSize = 50

// UpdateCursor is a position in a project's timeline: the (created_at, id)
// pair the order is built on. It carries the position itself rather than a
// reference to a row, so it stays valid after the row it was taken from is
// deleted (rv4-DATA-05).
type UpdateCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// Cursor is u's position in the timeline; pass it to ListProjectUpdatesBefore
// for the page that starts just after u.
func (u ProjectUpdate) Cursor() UpdateCursor {
	return UpdateCursor{CreatedAt: u.CreatedAt, ID: u.ID}
}

// ListProjectUpdates is ListProjectUpdatesBefore for a caller that only holds
// the id of the oldest row it has. The id is resolved to its position first;
// an id that is no longer a row of this project's timeline (deleted, or from
// another project) is ErrNotFound, not an empty page that reads as the end of
// history. Prefer ListProjectUpdatesBefore with ProjectUpdate.Cursor, which
// needs no lookup and survives the deletion.
func (s *Store) ListProjectUpdates(ctx context.Context, actorID, projectID uuid.UUID, limit int, before *uuid.UUID) ([]ProjectUpdate, error) {
	if before == nil {
		return s.ListProjectUpdatesBefore(ctx, actorID, projectID, limit, nil)
	}
	var at UpdateCursor
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			SELECT created_at, id FROM insideout.project_updates
			WHERE id = $1 AND project_id = $2`, *before, projectID,
		).Scan(&at.CreatedAt, &at.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.ListProjectUpdatesBefore(ctx, actorID, projectID, limit, &at)
}

// ListProjectUpdatesBefore requires the caller to have already verified
// membership (via GetProjectForMember), mirroring the ListMembers pattern.
// It returns one page of the timeline, newest first, capped at limit (<=0
// means ProjectUpdatesPageSize). `before` (optional) is the position of the
// oldest row the caller already has; the page starts just after it. Keyset
// pagination is on (created_at, id): created_at alone is ambiguous because rows
// written in one transaction share a statement-time timestamp (e.g. a GitHub
// sync batch), so the id breaks ties and keeps the cursor stable across pages.
func (s *Store) ListProjectUpdatesBefore(ctx context.Context, actorID, projectID uuid.UUID, limit int, before *UpdateCursor) ([]ProjectUpdate, error) {
	if limit <= 0 {
		limit = ProjectUpdatesPageSize
	}
	var beforeAt *time.Time
	var beforeID *uuid.UUID
	if before != nil {
		beforeAt, beforeID = &before.CreatedAt, &before.ID
	}
	var out []ProjectUpdate
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
				SELECT pu.id, pu.project_id, pu.author_id, pu.kind, pu.content, pu.created_at,
				       COALESCE(pd.decision, '')
				FROM insideout.project_updates pu
				LEFT JOIN insideout.proposal_decisions pd ON pd.update_id = pu.id
				WHERE pu.project_id = $1
				  AND ($2::timestamptz IS NULL OR (pu.created_at, pu.id) < ($2::timestamptz, $3::uuid))
				ORDER BY pu.created_at DESC, pu.id DESC
				LIMIT $4`, projectID, beforeAt, beforeID, limit,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var u ProjectUpdate
			if err := rows.Scan(&u.ID, &u.ProjectID, &u.AuthorID, &u.Kind, &u.Content, &u.CreatedAt, &u.Decision); err != nil {
				return err
			}
			out = append(out, u)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateProjectUpdate requires actorID to be the update's author or a
// workspace admin, and still a member of the project's workspace (SEC-11).
func (s *Store) UpdateProjectUpdate(ctx context.Context, actorID, updateID uuid.UUID, content string) (*ProjectUpdate, error) {
	var u ProjectUpdate
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var authorID, workspaceID uuid.UUID
		err := tx.QueryRow(ctx, `
			SELECT pu.author_id, p.workspace_id
			FROM insideout.project_updates pu
			JOIN insideout.projects p ON p.id = pu.project_id
			WHERE pu.id = $1`, updateID,
		).Scan(&authorID, &workspaceID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		role, err := requireMember(ctx, tx, workspaceID, actorID)
		if err != nil {
			return err
		}
		if authorID != actorID && role != "admin" {
			return ErrForbidden
		}

		err = tx.QueryRow(ctx, `
			UPDATE insideout.project_updates SET content = $2
			WHERE id = $1
			RETURNING id, project_id, author_id, kind, content, created_at`,
			updateID, content,
		).Scan(&u.ID, &u.ProjectID, &u.AuthorID, &u.Kind, &u.Content, &u.CreatedAt)
		// A concurrent delete between the SELECT above and this UPDATE removes
		// the row, so RETURNING yields nothing and Scan returns ErrNoRows — map
		// it to ErrNotFound (404) instead of leaking an opaque error as a 500,
		// mirroring the SELECT's own mapping just above (R4).
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// DeleteProjectUpdate requires actorID to be the update's author or a
// workspace admin, and still a member of the project's workspace (SEC-11).
func (s *Store) DeleteProjectUpdate(ctx context.Context, actorID, updateID uuid.UUID) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var authorID, workspaceID uuid.UUID
		err := tx.QueryRow(ctx, `
			SELECT pu.author_id, p.workspace_id
			FROM insideout.project_updates pu
			JOIN insideout.projects p ON p.id = pu.project_id
			WHERE pu.id = $1`, updateID,
		).Scan(&authorID, &workspaceID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		role, err := requireMember(ctx, tx, workspaceID, actorID)
		if err != nil {
			return err
		}
		if authorID != actorID && role != "admin" {
			return ErrForbidden
		}

		_, err = tx.Exec(ctx, `DELETE FROM insideout.project_updates WHERE id = $1`, updateID)
		return err
	})
}
