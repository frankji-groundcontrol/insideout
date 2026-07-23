package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Project struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Title       string
	Description string
	OwnerID     *uuid.UUID
	CreatedBy   uuid.UUID
	Status      string
	RepoURL     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProjectWithLatest is a project annotated with its most recent update (if
// any) — the board view's core data shape.
type ProjectWithLatest struct {
	Project
	LatestUpdateKind    *string
	LatestUpdateContent *string
	LatestUpdateAt      *time.Time
}

const projectColumns = `id, workspace_id, title, description, owner_id, created_by, status, repo_url, created_at, updated_at`

func scanProject(row pgx.Row) (*Project, error) {
	var p Project
	err := row.Scan(&p.ID, &p.WorkspaceID, &p.Title, &p.Description, &p.OwnerID, &p.CreatedBy, &p.Status, &p.RepoURL, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &p, err
}

// CreateProject requires actorID to currently be a workspace member,
// re-checked in the same transaction as the insert. Owner defaults to the
// creator.
func (s *Store) CreateProject(ctx context.Context, actorID, workspaceID uuid.UUID, title, description string) (*Project, error) {
	var p *Project
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if _, err := requireMember(ctx, tx, workspaceID, actorID); err != nil {
			return err
		}
		row := tx.QueryRow(ctx, `
			INSERT INTO insideout.projects (workspace_id, title, description, owner_id, created_by)
			VALUES ($1, $2, $3, $4, $4)
			RETURNING `+projectColumns,
			workspaceID, title, description, actorID,
		)
		var scanErr error
		p, scanErr = scanProject(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// ListProjectsForWorkspace requires actorID to be a member; a non-member
// gets an empty result via GetMembership check done by the caller (api
// layer calls GetMembership first and 404s), matching the pattern used
// for ListMembers.
func (s *Store) ListProjectsForWorkspace(ctx context.Context, actorID, workspaceID uuid.UUID) ([]ProjectWithLatest, error) {
	var out []ProjectWithLatest
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT `+qualifyColumns("p", projectColumns)+`,
			       lu.kind, lu.content, lu.created_at
			FROM insideout.projects p
			LEFT JOIN LATERAL (
				SELECT kind, content, created_at
				FROM insideout.project_updates
				WHERE project_id = p.id
				ORDER BY created_at DESC
				LIMIT 1
			) lu ON true
			WHERE p.workspace_id = $1
			ORDER BY COALESCE(lu.created_at, p.created_at) DESC`,
			workspaceID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var pw ProjectWithLatest
			if err := rows.Scan(&pw.ID, &pw.WorkspaceID, &pw.Title, &pw.Description, &pw.OwnerID, &pw.CreatedBy, &pw.Status, &pw.RepoURL, &pw.CreatedAt, &pw.UpdatedAt,
				&pw.LatestUpdateKind, &pw.LatestUpdateContent, &pw.LatestUpdateAt); err != nil {
				return err
			}
			out = append(out, pw)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetProjectForMember returns the project only if viewerID is a member of
// its workspace.
func (s *Store) GetProjectForMember(ctx context.Context, projectID, viewerID uuid.UUID) (*Project, error) {
	var p *Project
	err := s.withUserContext(ctx, viewerID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT `+qualifyColumns("p", projectColumns)+`
			FROM insideout.projects p
			JOIN insideout.workspace_memberships m ON m.workspace_id = p.workspace_id AND m.user_id = $2
			WHERE p.id = $1`,
			projectID, viewerID,
		)
		var scanErr error
		p, scanErr = scanProject(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

type ProjectUpdateFields struct {
	Title       string
	Description string
	Status      string
	OwnerID     *uuid.UUID
}

// UpdateProject requires actorID to be the project's owner or the
// workspace's admin, re-checked transactionally. No explicit row lock
// (FOR UPDATE) — see the comment on requireCreatorOrAdmin in
// workspaces.go for why: this table's SELECT policy references
// workspace_memberships, which Postgres cannot correctly re-evaluate
// under EvalPlanQual row-locking.
func (s *Store) UpdateProject(ctx context.Context, actorID, projectID uuid.UUID, f ProjectUpdateFields) (*Project, error) {
	var p *Project
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var workspaceID uuid.UUID
		var ownerID *uuid.UUID
		err := tx.QueryRow(ctx, `SELECT workspace_id, owner_id FROM insideout.projects WHERE id = $1`, projectID).
			Scan(&workspaceID, &ownerID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		isOwner := ownerID != nil && *ownerID == actorID
		if !isOwner {
			role, err := requireMember(ctx, tx, workspaceID, actorID)
			if err != nil {
				return err
			}
			if role != "admin" {
				return ErrForbidden
			}
		}

		row := tx.QueryRow(ctx, `
			UPDATE insideout.projects
			SET title = $2, description = $3, status = $4, owner_id = $5
			WHERE id = $1
			RETURNING `+projectColumns,
			projectID, f.Title, f.Description, f.Status, f.OwnerID,
		)
		var scanErr error
		p, scanErr = scanProject(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// DeleteProject requires actorID to be a workspace admin.
func (s *Store) DeleteProject(ctx context.Context, actorID, projectID uuid.UUID) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var workspaceID uuid.UUID
		err := tx.QueryRow(ctx, `SELECT workspace_id FROM insideout.projects WHERE id = $1`, projectID).Scan(&workspaceID)
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
		if role != "admin" {
			return ErrForbidden
		}

		_, err = tx.Exec(ctx, `DELETE FROM insideout.projects WHERE id = $1`, projectID)
		return err
	})
}
