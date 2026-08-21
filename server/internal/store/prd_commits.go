package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/prdcommit"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PrdCommit is one human-confirmed immutable PRD version (PRODUCT.md).
type PrdCommit struct {
	ID              uuid.UUID       `json:"id"`
	PrdID           uuid.UUID       `json:"prdId"`
	Revision        int             `json:"revision"`
	Name            string          `json:"name"`
	PrimaryAudience string          `json:"primaryAudience"`
	ChangeSummary   string          `json:"changeSummary"`
	Unresolved      json.RawMessage `json:"unresolved"`
	DecisionNote    string          `json:"decisionNote"`
	Diff            json.RawMessage `json:"diff"`
	CommittedBy     uuid.UUID       `json:"committedBy"`
	CommittedByName string          `json:"committedByName,omitempty"`
	CreatedAt       time.Time       `json:"createdAt"`
}

// CreatePrdCommit snapshots the PRD's current sections as a new
// revision (the existing revision machinery) and records the commit
// against it, with the section diff versus the previous commit. One
// transaction: the snapshot, the prds.current_revision bump, and the
// commit row land together or not at all.
func (s *Store) CreatePrdCommit(ctx context.Context, actorID, prdID uuid.UUID, name, audience, summary string, unresolved json.RawMessage, decisionNote string) (*PrdCommit, error) {
	var out *PrdCommit
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var workspaceID uuid.UUID
		var authorID uuid.UUID
		var current int
		var sections map[string]string
		if err := tx.QueryRow(ctx, `
			SELECT workspace_id, author_id, current_revision, sections
			FROM insideout.prds WHERE id = $1 FOR UPDATE`, prdID,
		).Scan(&workspaceID, &authorID, &current, &sections); err != nil {
			return err
		}
		if authorID != actorID {
			var admin bool
			if err := tx.QueryRow(ctx,
				`SELECT insideout._is_admin($1, $2)`, workspaceID, actorID).Scan(&admin); err != nil {
				return err
			}
			if !admin {
				return ErrForbidden
			}
		}

		// The previous commit's frozen sections, for the diff.
		var prevSections map[string]string
		if err := tx.QueryRow(ctx, `
			SELECT r.sections FROM insideout.prd_commits c
			JOIN insideout.prd_revisions r ON r.prd_id = c.prd_id AND r.revision = c.revision
			WHERE c.prd_id = $1
			ORDER BY c.created_at DESC, c.revision DESC LIMIT 1`, prdID,
		).Scan(&prevSections); err != nil && err != pgx.ErrNoRows {
			return err
		}
		diffRaw, err := json.Marshal(prdcommit.Diff(prevSections, sections))
		if err != nil {
			return err
		}
		diff := json.RawMessage(diffRaw)

		rev := current + 1
		if _, err := tx.Exec(ctx, `
			INSERT INTO insideout.prd_revisions (prd_id, revision, sections, created_by, note)
			VALUES ($1, $2, $3, $4, $5)`, prdID, rev, sections, actorID, "commit: "+name); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE insideout.prds SET current_revision = $2, updated_at = now() WHERE id = $1`, prdID, rev); err != nil {
			return err
		}
		if unresolved == nil {
			unresolved = json.RawMessage("[]")
		}
		var id uuid.UUID
		var created time.Time
		if err := tx.QueryRow(ctx, `
			INSERT INTO insideout.prd_commits
				(prd_id, revision, name, primary_audience, change_summary, unresolved, decision_note, diff, committed_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id, created_at`,
			prdID, rev, name, audience, summary, unresolved, decisionNote, diff, actorID,
		).Scan(&id, &created); err != nil {
			return err
		}
		out = &PrdCommit{
			ID: id, PrdID: prdID, Revision: rev, Name: name, PrimaryAudience: audience,
			ChangeSummary: summary, Unresolved: unresolved, DecisionNote: decisionNote,
			Diff: diff, CommittedBy: actorID, CreatedAt: created,
		}
		return nil
	})
	return out, err
}

// ListPrdCommits returns a PRD's version history, newest first, with
// committer names.
func (s *Store) ListPrdCommits(ctx context.Context, actorID, prdID uuid.UUID) ([]PrdCommit, error) {
	var out []PrdCommit
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT c.id, c.prd_id, c.revision, c.name, c.primary_audience,
			       c.change_summary, c.unresolved, c.decision_note, c.diff,
			       c.committed_by, COALESCE(u.username, ''), c.created_at
			FROM insideout.prd_commits c
			LEFT JOIN insideout.users u ON u.id = c.committed_by
			WHERE c.prd_id = $1
			ORDER BY c.revision DESC LIMIT 100`, prdID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var c PrdCommit
			if err := rows.Scan(&c.ID, &c.PrdID, &c.Revision, &c.Name, &c.PrimaryAudience,
				&c.ChangeSummary, &c.Unresolved, &c.DecisionNote, &c.Diff,
				&c.CommittedBy, &c.CommittedByName, &c.CreatedAt); err != nil {
				return err
			}
			out = append(out, c)
		}
		return rows.Err()
	})
	return out, err
}
