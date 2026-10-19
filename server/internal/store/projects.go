package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// ListProjectsForWorkspace requires actorID to be a member, re-checked in the
// same transaction as the read (ARCH-08, like ListMembers): a non-member gets
// ErrForbidden, not an empty board. The API still answers 404 up front via
// GetMembership.
func (s *Store) ListProjectsForWorkspace(ctx context.Context, actorID, workspaceID uuid.UUID) ([]ProjectWithLatest, error) {
	var out []ProjectWithLatest
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if _, err := requireMember(ctx, tx, workspaceID, actorID); err != nil {
			return err
		}

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

// ErrHandoverNeedsAdmin is the ErrForbidden a project's owner gets for one thing
// only: asking to give the project to someone else, or to nobody, without being
// a workspace admin. errors.Is(err, ErrForbidden) holds for it too.
var ErrHandoverNeedsAdmin = fmt.Errorf("%w: giving a project to someone else needs a workspace admin", ErrForbidden)

// UpdateProject requires actorID to be the project's owner or the
// workspace's admin, re-checked transactionally. Changing the owner is
// narrower: an owner who is not a workspace admin may edit the project but
// not give it away, and gets ErrHandoverNeedsAdmin before the UPDATE is tried
// (an edit that keeps the current owner is still the owner's). A new owner
// who is not a member of the workspace is ErrValidation. No explicit
// row lock (FOR UPDATE) — see the comment on requireCreatorOrAdmin in
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

		// The owner needs no other standing for an edit that keeps the owner;
		// any other edit, and any change of owner, needs an admin.
		isOwner := ownerID != nil && *ownerID == actorID
		keepsOwner := ownerID != nil && f.OwnerID != nil && *f.OwnerID == *ownerID
		if !isOwner || !keepsOwner {
			role, err := requireMember(ctx, tx, workspaceID, actorID)
			if err != nil {
				return err
			}
			if role != "admin" {
				if isOwner {
					return ErrHandoverNeedsAdmin
				}
				return ErrForbidden
			}
		}

		// Whom the project is given to has to be a member of its workspace, as a
		// roadmap node's owner does. Judged after the caller's standing, so a
		// caller who may not change the owner learns nothing about who is in
		// the workspace; and only for a change, so a project whose owner has
		// since left stays editable with the owner the client was shown.
		if f.OwnerID != nil && !keepsOwner {
			_, err := requireMember(ctx, tx, workspaceID, *f.OwnerID)
			if errors.Is(err, ErrForbidden) {
				return ErrValidation
			}
			if err != nil {
				return err
			}
		}

		var upErr error
		p, upErr = updateProjectRow(ctx, tx, projectID, f)
		return upErr
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// updateProjectRow is the one UPDATE of a project's editable fields. Who may
// call it is decided by UpdateProject before it gets here. The projects_update
// policy has a USING clause and no WITH CHECK, so Postgres judges the new row
// by USING too and refuses an owner who is not an admin and changes owner_id
// with 42501; UpdateProject spares the caller that, and this maps it to
// ErrForbidden for a caller whose standing changed after the check.
func updateProjectRow(ctx context.Context, tx pgx.Tx, projectID uuid.UUID, f ProjectUpdateFields) (*Project, error) {
	p, err := scanProject(tx.QueryRow(ctx, `
		UPDATE insideout.projects
		SET title = $2, description = $3, status = $4, owner_id = $5
		WHERE id = $1
		RETURNING `+projectColumns,
		projectID, f.Title, f.Description, f.Status, f.OwnerID,
	))
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42501" {
		return nil, ErrForbidden
	}
	return p, err
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
