package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// requireProjectOwnerOrAdmin allows the project owner or a workspace admin —
// the level the projects_update RLS policy requires before the projects row
// (repo link, sync cursor in meta) may change.
func requireProjectOwnerOrAdmin(ctx context.Context, tx pgx.Tx, projectID, actorID uuid.UUID) error {
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
	if ownerID != nil && *ownerID == actorID {
		return nil
	}
	role, err := requireMember(ctx, tx, workspaceID, actorID)
	if err != nil {
		return err
	}
	if role != "admin" {
		return ErrForbidden
	}
	return nil
}

// SetProjectRepo links (or clears) the GitHub repo a project syncs from.
func (s *Store) SetProjectRepo(ctx context.Context, actorID, projectID uuid.UUID, repoURL string) (*Project, error) {
	var p *Project
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if err := requireProjectOwnerOrAdmin(ctx, tx, projectID, actorID); err != nil {
			return err
		}
		row := tx.QueryRow(ctx, `
			UPDATE insideout.projects SET repo_url = $2 WHERE id = $1 RETURNING `+projectColumns,
			projectID, repoURL,
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

// ProjectRepoSync reads the linked repo URL and the last-synced commit SHA.
func (s *Store) ProjectRepoSync(ctx context.Context, actorID, projectID uuid.UUID) (repoURL, lastSHA string, err error) {
	err = s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if err := requireProjectOwnerOrAdmin(ctx, tx, projectID, actorID); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `
			SELECT repo_url, COALESCE(meta->>'github_last_sha', '') FROM insideout.projects WHERE id = $1`,
			projectID,
		).Scan(&repoURL, &lastSHA)
	})
	return repoURL, lastSHA, err
}

// RecordRepoSyncSHA advances the sync cursor to the newest commit SHA seen.
func (s *Store) RecordRepoSyncSHA(ctx context.Context, actorID, projectID uuid.UUID, sha string) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if err := requireProjectOwnerOrAdmin(ctx, tx, projectID, actorID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE insideout.projects
			SET meta = jsonb_set(meta, '{github_last_sha}', to_jsonb($2::text))
			WHERE id = $1`,
			projectID, sha,
		)
		return err
	})
}
