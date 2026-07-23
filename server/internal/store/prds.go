package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PrdSectionKeys are the 8 fixed PRD sections per
// docs/plans/2026-07-20-go-rewrite/03-agents.md §2. Order matters for
// rendering; validity of a section key is checked against this list.
var PrdSectionKeys = []string{
	"background", "users", "goals", "nonGoals",
	"stories", "requirements", "constraints", "risks",
}

func emptySections() map[string]string {
	m := make(map[string]string, len(PrdSectionKeys))
	for _, k := range PrdSectionKeys {
		m[k] = ""
	}
	return m
}

type Prd struct {
	ID              uuid.UUID
	WorkspaceID     uuid.UUID
	IdeaID          *uuid.UUID
	ProjectID       *uuid.UUID
	AuthorID        uuid.UUID
	Title           string
	Sections        map[string]string
	Status          string
	CurrentRevision int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

const prdColumns = `id, workspace_id, idea_id, project_id, author_id, title, sections, status, current_revision, created_at, updated_at`

func scanPrd(row pgx.Row) (*Prd, error) {
	var p Prd
	err := row.Scan(&p.ID, &p.WorkspaceID, &p.IdeaID, &p.ProjectID, &p.AuthorID, &p.Title, &p.Sections, &p.Status, &p.CurrentRevision, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &p, err
}

// insertPrd is used both by ConvertIdea (idea.go) and could be reused for
// a future "start a PRD without an idea" path; it must be called with an
// open transaction.
//
// sections is marshaled to JSON explicitly, bound as a Go string (not
// []byte), with an explicit ::jsonb cast — this pattern applies
// everywhere a Go map/struct is written to a jsonb column. Two separate
// pgx quirks under the simple query protocol (used for pgbouncer
// transaction-pooling connections, see internal/store/pool.go) force
// this: (1) pgx has no default encoder for arbitrary map types at all,
// erroring "cannot find encode plan"; (2) even for a pre-marshaled
// []byte, pgx's default encoding is bytea (hex-escaped), and casting
// that hex text as ::jsonb fails — Postgres tries to parse the escaped
// hex string itself as JSON. Only a Go string gets embedded as a plain,
// unescaped text literal, which ::jsonb can actually parse.
func insertPrd(ctx context.Context, tx pgx.Tx, workspaceID uuid.UUID, ideaID *uuid.UUID, authorID uuid.UUID, title string) (*Prd, error) {
	sectionsJSON, err := json.Marshal(emptySections())
	if err != nil {
		return nil, err
	}
	row := tx.QueryRow(ctx, `
		INSERT INTO insideout.prds (workspace_id, idea_id, author_id, title, sections)
		VALUES ($1, $2, $3, $4, $5::jsonb)
		RETURNING `+prdColumns,
		workspaceID, ideaID, authorID, title, string(sectionsJSON),
	)
	return scanPrd(row)
}

// GetPrdForMember returns the PRD only if viewerID is a member of its
// workspace.
func (s *Store) GetPrdForMember(ctx context.Context, prdID, viewerID uuid.UUID) (*Prd, error) {
	var prd *Prd
	err := s.withUserContext(ctx, viewerID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT `+qualifyColumns("p", prdColumns)+`
			FROM insideout.prds p
			JOIN insideout.workspace_memberships m ON m.workspace_id = p.workspace_id AND m.user_id = $2
			WHERE p.id = $1`,
			prdID, viewerID,
		)
		var scanErr error
		prd, scanErr = scanPrd(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return prd, nil
}

// UpdateSections requires actorID to be the PRD's author or a workspace
// admin. Only keys present in patch are changed; unknown keys are the
// caller's responsibility to validate against PrdSectionKeys before
// calling (kept here so store stays a thin, honest persistence layer).
//
// expectedUpdatedAt, when non-nil, is a compare-and-swap check against
// the PRD's current updated_at: if it doesn't match, ErrConflict is
// returned and nothing is written — this is how the coach's
// update_prd_section tool avoids silently clobbering a concurrent human
// edit (docs/plans/2026-07-21-prd-agent-harness/plan.md §5.4). Manual
// saves via the PATCH endpoint pass nil: no CAS, last-write-wins, which
// is what makes a human's direct edit always win over a stale coach
// write rather than the reverse.
func (s *Store) UpdateSections(ctx context.Context, actorID, prdID uuid.UUID, title string, patch map[string]string, expectedUpdatedAt *time.Time) (*Prd, error) {
	var updated *Prd
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		prd, err := requirePrdAuthorOrAdmin(ctx, tx, prdID, actorID)
		if err != nil {
			return err
		}
		if expectedUpdatedAt != nil && !prd.UpdatedAt.Equal(*expectedUpdatedAt) {
			return ErrConflict
		}

		merged := prd.Sections
		if merged == nil {
			merged = emptySections()
		}
		for k, v := range patch {
			merged[k] = v
		}
		mergedJSON, err := json.Marshal(merged)
		if err != nil {
			return err
		}

		row := tx.QueryRow(ctx, `
			UPDATE insideout.prds SET title = $2, sections = $3::jsonb
			WHERE id = $1
			RETURNING `+prdColumns,
			prdID, title, string(mergedJSON),
		)
		var scanErr error
		updated, scanErr = scanPrd(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// requirePrdAuthorOrAdmin locks and returns the PRD row if actorID is its
// author or the workspace's admin; must run inside an open transaction.
func requirePrdAuthorOrAdmin(ctx context.Context, tx pgx.Tx, prdID, actorID uuid.UUID) (*Prd, error) {
	row := tx.QueryRow(ctx, `SELECT `+prdColumns+` FROM insideout.prds WHERE id = $1`, prdID)
	prd, err := scanPrd(row)
	if err != nil {
		return nil, err
	}
	if prd.AuthorID == actorID {
		return prd, nil
	}
	role, err := requireMember(ctx, tx, prd.WorkspaceID, actorID)
	if err != nil {
		return nil, err
	}
	if role != "admin" {
		return nil, ErrForbidden
	}
	return prd, nil
}

var prdTransitions = map[string][]string{
	"draft":     {"reviewing"},
	"reviewing": {"approved", "rejected"},
	"rejected":  {"draft"},
	"approved":  {},
}

// UpdateStatus enforces the PRD review lifecycle
// (draft->reviewing->approved/rejected, rejected->draft) and its
// authorization split: the author submits and resubmits; only a
// workspace admin approves or rejects (never the author acting alone).
func (s *Store) UpdatePrdStatus(ctx context.Context, actorID, prdID uuid.UUID, newStatus string) (*Prd, error) {
	var updated *Prd
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `SELECT `+prdColumns+` FROM insideout.prds WHERE id = $1`, prdID)
		prd, err := scanPrd(row)
		if err != nil {
			return err
		}

		allowed := false
		for _, next := range prdTransitions[prd.Status] {
			if next == newStatus {
				allowed = true
				break
			}
		}
		if !allowed {
			return ErrValidation
		}

		authorOnly := newStatus == "reviewing" || newStatus == "draft"
		if authorOnly {
			if prd.AuthorID != actorID {
				return ErrForbidden
			}
		} else {
			role, err := requireMember(ctx, tx, prd.WorkspaceID, actorID)
			if err != nil {
				return err
			}
			if role != "admin" {
				return ErrForbidden
			}
		}

		updatedRow := tx.QueryRow(ctx, `
			UPDATE insideout.prds SET status = $2 WHERE id = $1
			RETURNING `+prdColumns, prdID, newStatus,
		)
		var scanErr error
		updated, scanErr = scanPrd(updatedRow)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}
