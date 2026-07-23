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

// ListProjectUpdates requires the caller to have already verified
// membership (via GetProjectForMember), mirroring the ListMembers pattern.
func (s *Store) ListProjectUpdates(ctx context.Context, actorID, projectID uuid.UUID) ([]ProjectUpdate, error) {
	var out []ProjectUpdate
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, project_id, author_id, kind, content, created_at
			FROM insideout.project_updates
			WHERE project_id = $1
			ORDER BY created_at DESC`, projectID,
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

		return tx.QueryRow(ctx, `
			UPDATE insideout.project_updates SET content = $2
			WHERE id = $1
			RETURNING id, project_id, author_id, kind, content, created_at`,
			updateID, content,
		).Scan(&u.ID, &u.ProjectID, &u.AuthorID, &u.Kind, &u.Content, &u.CreatedAt)
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
