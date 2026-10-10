package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PrdRevision struct {
	ID        uuid.UUID
	PrdID     uuid.UUID
	Revision  int
	Sections  map[string]string
	CreatedBy uuid.UUID
	Note      *string
	CreatedAt time.Time
}

// lockPrdRevisions queues every writer of one PRD's revision numbers behind a
// per-PRD advisory lock held to the end of the transaction. Without it,
// concurrent writers all read the same current_revision, all compute +1, and
// the losers trip UNIQUE(prd_id, revision) — a spurious 409 — or, when one
// holds the PRD row and another holds the pending insert, deadlock. Every
// writer takes this lock before anything else. The key is namespaced: roadmap
// builds lock the bare hashtext(prdID), and the two must not block each other.
func lockPrdRevisions(ctx context.Context, tx pgx.Tx, prdID uuid.UUID) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "prd_revision:"+prdID.String())
	return err
}

// CreateRevision snapshots the PRD's current sections as the next
// revision number (the same MAX+1 pattern the old complete_node RPC used
// for document_revisions — see docs/plans/2026-07-20-go-rewrite/01-database.md §5).
// Concurrent snapshots of one PRD queue behind an advisory lock, not a row
// lock (FOR UPDATE) — see the comment on requireCreatorOrAdmin in
// workspaces.go for why. Requires actorID to be the PRD's author.
func (s *Store) CreateRevision(ctx context.Context, actorID, prdID uuid.UUID, note *string) (*PrdRevision, error) {
	var rev PrdRevision
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if err := lockPrdRevisions(ctx, tx, prdID); err != nil {
			return err
		}

		var authorID uuid.UUID
		var sections map[string]string
		var currentRevision int
		err := tx.QueryRow(ctx, `
			SELECT author_id, sections, current_revision
			FROM insideout.prds WHERE id = $1`, prdID,
		).Scan(&authorID, &sections, &currentRevision)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if authorID != actorID {
			return ErrForbidden
		}

		sectionsJSON, err := json.Marshal(sections)
		if err != nil {
			return err
		}

		nextRevision := currentRevision + 1
		err = tx.QueryRow(ctx, `
			INSERT INTO insideout.prd_revisions (prd_id, revision, sections, created_by, note)
			VALUES ($1, $2, $3::jsonb, $4, $5)
			RETURNING id, prd_id, revision, sections, created_by, note, created_at`,
			prdID, nextRevision, string(sectionsJSON), actorID, note,
		).Scan(&rev.ID, &rev.PrdID, &rev.Revision, &rev.Sections, &rev.CreatedBy, &rev.Note, &rev.CreatedAt)
		// The lock above queues revision writers, so they no longer collide here.
		// It stays as a backstop for any writer that skips the lock: map the
		// (prd_id, revision) unique violation (23505) to ErrConflict so the API
		// answers 409 "try again" instead of an opaque 500 (F14).
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConflict
		}
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `UPDATE insideout.prds SET current_revision = $2 WHERE id = $1`, prdID, nextRevision)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &rev, nil
}

// ListRevisions requires the caller to have already verified membership
// (via GetPrdForMember).
func (s *Store) ListRevisions(ctx context.Context, actorID, prdID uuid.UUID) ([]PrdRevision, error) {
	var out []PrdRevision
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, prd_id, revision, sections, created_by, note, created_at
			FROM insideout.prd_revisions
			WHERE prd_id = $1
			ORDER BY revision DESC`, prdID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var rev PrdRevision
			if err := rows.Scan(&rev.ID, &rev.PrdID, &rev.Revision, &rev.Sections, &rev.CreatedBy, &rev.Note, &rev.CreatedAt); err != nil {
				return err
			}
			out = append(out, rev)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
