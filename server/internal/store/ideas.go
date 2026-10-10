package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Idea struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	AuthorID    uuid.UUID
	Title       string
	Content     string
	Status      string
	PrdID       *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const ideaColumns = `id, workspace_id, author_id, title, content, status, prd_id, created_at, updated_at`

func scanIdea(row pgx.Row) (*Idea, error) {
	var i Idea
	err := row.Scan(&i.ID, &i.WorkspaceID, &i.AuthorID, &i.Title, &i.Content, &i.Status, &i.PrdID, &i.CreatedAt, &i.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &i, err
}

// CreateIdea requires actorID to currently be a workspace member.
func (s *Store) CreateIdea(ctx context.Context, actorID, workspaceID uuid.UUID, title, content string) (*Idea, error) {
	var idea *Idea
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if _, err := requireMember(ctx, tx, workspaceID, actorID); err != nil {
			return err
		}
		row := tx.QueryRow(ctx, `
			INSERT INTO insideout.ideas (workspace_id, author_id, title, content)
			VALUES ($1, $2, $3, $4)
			RETURNING `+ideaColumns,
			workspaceID, actorID, title, content,
		)
		var scanErr error
		idea, scanErr = scanIdea(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return idea, nil
}

// ListIdeas returns only the actor's own ideas — the inbox is private by
// default (USER-05; admins can still fetch a specific idea by ID, which
// matches their drop authority). Dropped ideas are a soft terminal state:
// the row stays for audit, but the wall is a working surface and stops
// listing them (USER-09). Requires actorID to be a member, re-checked in the
// same transaction as the read (ARCH-08): a non-member gets ErrForbidden, not
// an empty list. The API still answers 404 up front via GetMembership.
func (s *Store) ListIdeas(ctx context.Context, actorID, workspaceID uuid.UUID) ([]Idea, error) {
	var out []Idea
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if _, err := requireMember(ctx, tx, workspaceID, actorID); err != nil {
			return err
		}

		rows, err := tx.Query(ctx, `
			SELECT `+ideaColumns+`
			FROM insideout.ideas
			WHERE workspace_id = $1 AND author_id = $2 AND status <> 'dropped'
			ORDER BY created_at DESC`, workspaceID, actorID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var i Idea
			if err := rows.Scan(&i.ID, &i.WorkspaceID, &i.AuthorID, &i.Title, &i.Content, &i.Status, &i.PrdID, &i.CreatedAt, &i.UpdatedAt); err != nil {
				return err
			}
			out = append(out, i)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetIdeaForMember returns the idea only to its author or the workspace's
// admin (USER-05 private-by-default; the admin keeps read access to match
// their drop authority). Fellow members get ErrNotFound — existence is
// not disclosed.
func (s *Store) GetIdeaForMember(ctx context.Context, ideaID, viewerID uuid.UUID) (*Idea, error) {
	var idea *Idea
	err := s.withUserContext(ctx, viewerID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT `+qualifyColumns("i", ideaColumns)+`
			FROM insideout.ideas i
			WHERE i.id = $1
			  AND (i.author_id = $2
			       OR EXISTS (SELECT 1 FROM insideout.workspace_memberships m
			                  WHERE m.workspace_id = i.workspace_id AND m.user_id = $2 AND m.role = 'admin'))`,
			ideaID, viewerID,
		)
		var scanErr error
		idea, scanErr = scanIdea(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return idea, nil
}

// UpdateIdea requires actorID to be the idea's author.
func (s *Store) UpdateIdea(ctx context.Context, actorID, ideaID uuid.UUID, title, content string) (*Idea, error) {
	var idea *Idea
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var authorID uuid.UUID
		err := tx.QueryRow(ctx, `SELECT author_id FROM insideout.ideas WHERE id = $1`, ideaID).Scan(&authorID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if authorID != actorID {
			return ErrForbidden
		}

		row := tx.QueryRow(ctx, `
			UPDATE insideout.ideas
			SET title = $2, content = $3, status = CASE WHEN status = 'inbox' THEN 'refining' ELSE status END
			WHERE id = $1
			RETURNING `+ideaColumns,
			ideaID, title, content,
		)
		var scanErr error
		idea, scanErr = scanIdea(row)
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return idea, nil
}

// DropIdea marks the idea 'dropped' (soft terminal state, matching the
// lifecycle in docs/plans/2026-07-20-go-rewrite/README.md §4). Requires
// actorID to be the author or a workspace admin.
func (s *Store) DropIdea(ctx context.Context, actorID, ideaID uuid.UUID) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var authorID, workspaceID uuid.UUID
		err := tx.QueryRow(ctx, `SELECT author_id, workspace_id FROM insideout.ideas WHERE id = $1`, ideaID).
			Scan(&authorID, &workspaceID)
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

		_, err = tx.Exec(ctx, `UPDATE insideout.ideas SET status = 'dropped' WHERE id = $1`, ideaID)
		return err
	})
}

// ConvertIdea creates a PRD (with the fixed 8 empty sections) and its
// coaching conversation from an idea, atomically, and marks the idea
// 'converted'. Requires actorID to be the idea's author.
func (s *Store) ConvertIdea(ctx context.Context, actorID, ideaID uuid.UUID) (*Prd, *AgentConversation, error) {
	var prd *Prd
	var conv *AgentConversation
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var idea Idea
		err := tx.QueryRow(ctx, `SELECT `+ideaColumns+` FROM insideout.ideas WHERE id = $1`, ideaID).
			Scan(&idea.ID, &idea.WorkspaceID, &idea.AuthorID, &idea.Title, &idea.Content, &idea.Status, &idea.PrdID, &idea.CreatedAt, &idea.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if idea.AuthorID != actorID {
			return ErrForbidden
		}

		// Claim the conversion atomically BEFORE inserting anything. The plain
		// SELECT above holds no lock, so two concurrent converters both read
		// status='pending' and both pass a read-then-check guard — each would
		// insert its own PRD + conversation and commit, orphaning one (the
		// idea's prd_id can only point at one). The guard therefore lives in
		// this UPDATE: only a row still <> 'converted' is claimed, and because
		// concurrent UPDATEs on one row serialize, exactly one tx sees
		// RowsAffected==1 and the rest get 0 → ErrConflict (409). FOR UPDATE is
		// NOT an option here: the ideas SELECT policy joins
		// workspace_memberships, which returns zero rows under EvalPlanQual
		// re-evaluation and would silently break the lock (R1).
		tag, err := tx.Exec(ctx, `UPDATE insideout.ideas SET status = 'converted' WHERE id = $1 AND status <> 'converted'`, ideaID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrConflict
		}

		prd, err = insertPrd(ctx, tx, idea.WorkspaceID, &idea.ID, actorID, idea.Title)
		if err != nil {
			return err
		}

		conv, err = insertAgentConversation(ctx, tx, idea.WorkspaceID, actorID, prd.ID)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `UPDATE insideout.ideas SET prd_id = $2 WHERE id = $1`, ideaID, prd.ID)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	return prd, conv, nil
}
