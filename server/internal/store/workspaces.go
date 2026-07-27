package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Workspace struct {
	ID          uuid.UUID
	Title       string
	Description string
	CoverURL    *string
	Code        string
	CreatorID   uuid.UUID
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WorkspaceSummary is a workspace annotated with the viewer's own role in
// it and the total member count — the shape the dashboard's "joined" /
// "managed" lists need.
type WorkspaceSummary struct {
	Workspace
	MemberCount int
	MyRole      string
}

// generateInviteCode returns the workspace join credential. The code is the
// SOLE gate on joining a workspace and the join endpoint is authenticated but
// not rate-limited, so it must be unguessable: 128 bits from crypto/rand,
// hex-encoded (32 chars). The old `%06d` crushed those random bytes to a
// 10^6 keyspace that was brute-forceable (R2). The `code` column is `text`,
// so the longer code fits unchanged, and the collision-retry loop in
// createWorkspaceTx still guards uniqueness (collisions are now negligible).
func generateInviteCode() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

const maxCodeAttempts = 8

// CreateWorkspace creates a workspace with a collision-checked 128-bit
// invite code (fixing the old RPC's no-retry bug — see
// docs/plans/2026-07-20-go-rewrite/02-backend-go.md §3) and makes the
// creator its first admin member, atomically.
func (s *Store) CreateWorkspace(ctx context.Context, creatorID uuid.UUID, title, description string) (*Workspace, error) {
	var ws Workspace
	err := s.withUserContext(ctx, creatorID, func(tx pgx.Tx) error {
		return createWorkspaceTx(ctx, tx, creatorID, title, description, &ws)
	})
	if err != nil {
		return nil, err
	}
	return &ws, nil
}

func createWorkspaceTx(ctx context.Context, tx pgx.Tx, creatorID uuid.UUID, title, description string, ws *Workspace) error {
	for attempt := 0; attempt < maxCodeAttempts; attempt++ {
		code, err := generateInviteCode()
		if err != nil {
			return err
		}

		err = tx.QueryRow(ctx, `
			INSERT INTO insideout.workspaces (title, description, code, creator_id)
			VALUES ($1, $2, $3, $4)
			RETURNING id, title, description, cover_url, code, creator_id, status, created_at, updated_at`,
			title, description, code, creatorID,
		).Scan(&ws.ID, &ws.Title, &ws.Description, &ws.CoverURL, &ws.Code, &ws.CreatorID, &ws.Status, &ws.CreatedAt, &ws.UpdatedAt)

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			continue // code collision, retry with a new code / 邀请码冲突，换码重试
		}
		if err != nil {
			return err
		}
		break
	}
	if ws.ID == uuid.Nil {
		return fmt.Errorf("store: could not allocate a unique invite code after %d attempts", maxCodeAttempts)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO insideout.workspace_memberships (workspace_id, user_id, role)
		VALUES ($1, $2, 'admin')`, ws.ID, creatorID,
	); err != nil {
		return err
	}
	return nil
}

// JoinWorkspace adds userID as a member of the active workspace identified
// by code. A duplicate join surfaces as ErrConflict (409).
func (s *Store) JoinWorkspace(ctx context.Context, userID uuid.UUID, code string) (*Workspace, error) {
	var ws Workspace
	err := s.withUserContext(ctx, userID, func(tx pgx.Tx) error {
		// See the workspaces_select policy comment in
		// db/migrations/20260720150000_row_level_security.sql for why
		// this second session var exists.
		if _, err := tx.Exec(ctx, `SELECT set_config('app.join_code', $1, true)`, code); err != nil {
			return err
		}

		err := tx.QueryRow(ctx, `
			SELECT id, title, description, cover_url, code, creator_id, status, created_at, updated_at
			FROM insideout.workspaces WHERE code = $1 AND status = 'active'`, code,
		).Scan(&ws.ID, &ws.Title, &ws.Description, &ws.CoverURL, &ws.Code, &ws.CreatorID, &ws.Status, &ws.CreatedAt, &ws.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO insideout.workspace_memberships (workspace_id, user_id, role)
			VALUES ($1, $2, 'member')`, ws.ID, userID,
		)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConflict
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return &ws, nil
}

// GetWorkspaceForMember returns the workspace only if viewerID is
// currently a member — a non-member gets ErrNotFound, matching the old
// RLS "select member" rule (existence itself is not disclosed).
func (s *Store) GetWorkspaceForMember(ctx context.Context, workspaceID, viewerID uuid.UUID) (*WorkspaceSummary, error) {
	var ws WorkspaceSummary
	err := s.withUserContext(ctx, viewerID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT w.id, w.title, w.description, w.cover_url, w.code, w.creator_id, w.status, w.created_at, w.updated_at,
			       (SELECT count(*) FROM insideout.workspace_memberships m2 WHERE m2.workspace_id = w.id),
			       m.role
			FROM insideout.workspaces w
			JOIN insideout.workspace_memberships m ON m.workspace_id = w.id AND m.user_id = $2
			WHERE w.id = $1`,
			workspaceID, viewerID,
		).Scan(&ws.ID, &ws.Title, &ws.Description, &ws.CoverURL, &ws.Code, &ws.CreatorID, &ws.Status, &ws.CreatedAt, &ws.UpdatedAt,
			&ws.MemberCount, &ws.MyRole)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ws, nil
}

func (s *Store) ListWorkspacesForUser(ctx context.Context, userID uuid.UUID) ([]WorkspaceSummary, error) {
	var out []WorkspaceSummary
	err := s.withUserContext(ctx, userID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT w.id, w.title, w.description, w.cover_url, w.code, w.creator_id, w.status, w.created_at, w.updated_at,
			       (SELECT count(*) FROM insideout.workspace_memberships m2 WHERE m2.workspace_id = w.id),
			       m.role
			FROM insideout.workspaces w
			JOIN insideout.workspace_memberships m ON m.workspace_id = w.id
			WHERE m.user_id = $1
			ORDER BY w.updated_at DESC`, userID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var ws WorkspaceSummary
			if err := rows.Scan(&ws.ID, &ws.Title, &ws.Description, &ws.CoverURL, &ws.Code, &ws.CreatorID, &ws.Status, &ws.CreatedAt, &ws.UpdatedAt,
				&ws.MemberCount, &ws.MyRole); err != nil {
				return err
			}
			out = append(out, ws)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateWorkspace requires actorID to be the creator or a workspace admin,
// re-checked inside the same transaction as the write per the TOCTOU
// lesson in docs/plans/2026-07-20-go-rewrite/01-database.md §5.
func (s *Store) UpdateWorkspace(ctx context.Context, actorID, workspaceID uuid.UUID, title, description string, coverURL *string) (*Workspace, error) {
	var ws Workspace
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if err := requireCreatorOrAdmin(ctx, tx, workspaceID, actorID); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `
			UPDATE insideout.workspaces
			SET title = $2, description = $3, cover_url = $4
			WHERE id = $1
			RETURNING id, title, description, cover_url, code, creator_id, status, created_at, updated_at`,
			workspaceID, title, description, coverURL,
		).Scan(&ws.ID, &ws.Title, &ws.Description, &ws.CoverURL, &ws.Code, &ws.CreatorID, &ws.Status, &ws.CreatedAt, &ws.UpdatedAt)
	})
	if err != nil {
		return nil, err
	}
	return &ws, nil
}

// DeleteWorkspace requires actorID to be the creator.
func (s *Store) DeleteWorkspace(ctx context.Context, actorID, workspaceID uuid.UUID) error {
	return s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM insideout.workspaces WHERE id = $1 AND creator_id = $2`, workspaceID, actorID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrForbidden
		}
		return nil
	})
}

// requireCreatorOrAdmin must be called with an open transaction so the
// membership check and the subsequent write happen in the same
// transaction (the TOCTOU lesson in
// docs/plans/2026-07-20-go-rewrite/01-database.md §5). No explicit row
// lock (FOR KEY SHARE) here — Postgres cannot correctly re-evaluate a
// FORCE'd RLS policy that references another table under EvalPlanQual
// row-locking (confirmed empirically: it silently returns zero rows,
// not an error), and this table's SELECT policy references
// workspace_memberships. `// ponytail: narrower TOCTOU window than an
// explicit lock would give (a concurrent role change between this check
// and the write could theoretically slip through) — upgrade path is
// embedding the same condition directly in the UPDATE's WHERE clause if
// this race ever matters in practice.`
func requireCreatorOrAdmin(ctx context.Context, tx pgx.Tx, workspaceID, actorID uuid.UUID) error {
	var role *string
	var creatorID uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT w.creator_id, m.role
		FROM insideout.workspaces w
		LEFT JOIN insideout.workspace_memberships m ON m.workspace_id = w.id AND m.user_id = $2
		WHERE w.id = $1`,
		workspaceID, actorID,
	).Scan(&creatorID, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if creatorID == actorID {
		return nil
	}
	if role != nil && *role == "admin" {
		return nil
	}
	return ErrForbidden
}
