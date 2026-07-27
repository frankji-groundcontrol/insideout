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
}

// AddProjectUpdate requires actorID to currently be a member of the
// project's workspace.
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

// ListProjectUpdates requires the caller to have already verified
// membership (via GetProjectForMember), mirroring the ListMembers pattern.
// It returns one page of the timeline, newest first, capped at limit (<=0
// means ProjectUpdatesPageSize). `before` (optional) is the id of the oldest
// row the caller already has; the page starts just after it. Keyset pagination
// is on (created_at, id): created_at alone is ambiguous because rows written in
// one transaction share a statement-time timestamp (e.g. a GitHub sync batch),
// so the id breaks ties and keeps the cursor stable across pages.
func (s *Store) ListProjectUpdates(ctx context.Context, actorID, projectID uuid.UUID, limit int, before *uuid.UUID) ([]ProjectUpdate, error) {
	if limit <= 0 {
		limit = ProjectUpdatesPageSize
	}
	var out []ProjectUpdate
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, project_id, author_id, kind, content, created_at
			FROM insideout.project_updates
			WHERE project_id = $1
			  AND ($2::uuid IS NULL OR (created_at, id) < (
			    SELECT created_at, id FROM insideout.project_updates WHERE id = $2
			  ))
			ORDER BY created_at DESC, id DESC
			LIMIT $3`, projectID, before, limit,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var u ProjectUpdate
			if err := rows.Scan(&u.ID, &u.ProjectID, &u.AuthorID, &u.Kind, &u.Content, &u.CreatedAt); err != nil {
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
// workspace admin.
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

		if authorID != actorID {
			role, err := requireMember(ctx, tx, workspaceID, actorID)
			if err != nil {
				return err
			}
			if role != "admin" {
				return ErrForbidden
			}
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
// workspace admin.
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

		if authorID != actorID {
			role, err := requireMember(ctx, tx, workspaceID, actorID)
			if err != nil {
				return err
			}
			if role != "admin" {
				return ErrForbidden
			}
		}

		_, err = tx.Exec(ctx, `DELETE FROM insideout.project_updates WHERE id = $1`, updateID)
		return err
	})
}
