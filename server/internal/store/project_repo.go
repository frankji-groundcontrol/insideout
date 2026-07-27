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

// SyncRepoCommits appends synced GitHub commits to a project's progress
// timeline AND advances the sync cursor in ONE transaction. contents are
// inserted in the order given (caller orders them oldest-first so the
// timeline reads chronologically); newestSHA becomes the new cursor.
//
// Doing both in a single transaction is the fix for duplicate timeline
// entries: previously each commit was its own transaction followed by a
// separate cursor write, so a crash (or any error) between the inserts and
// the cursor advance left commits written with the cursor still behind them —
// and the next sync re-inserted the same commits. Now either the whole batch
// plus the cursor land, or nothing does, so a re-sync after a failure retries
// cleanly. The owner/admin check (requireProjectOwnerOrAdmin) matches what
// ProjectRepoSync already enforced on the read side of the sync.
func (s *Store) SyncRepoCommits(ctx context.Context, actorID, projectID uuid.UUID, contents []string, newestSHA string) (int, error) {
	if len(contents) == 0 {
		return 0, nil // nothing new: leave the cursor exactly where it is
	}
	added := 0
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if err := requireProjectOwnerOrAdmin(ctx, tx, projectID, actorID); err != nil {
			return err
		}
		for _, content := range contents {
			if _, err := tx.Exec(ctx, `
				INSERT INTO insideout.project_updates (project_id, author_id, kind, content)
				VALUES ($1, $2, 'progress', $3)`,
				projectID, actorID, content,
			); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `
			UPDATE insideout.projects
			SET meta = jsonb_set(meta, '{github_last_sha}', to_jsonb($2::text))
			WHERE id = $1`,
			projectID, newestSHA,
		); err != nil {
			return err
		}
		added = len(contents)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return added, nil
}
