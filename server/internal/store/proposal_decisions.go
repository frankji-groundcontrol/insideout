package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ProposalDecision is the human decision on an agent proposal — the
// Decision Log entry (immutable; re-deciding appends, see
// DecideProposal's timeline note).
type ProposalDecision struct {
	UpdateID  uuid.UUID `json:"updateId"`
	Decision  string    `json:"decision"`
	Reason    string    `json:"reason"`
	DecidedBy uuid.UUID `json:"decidedBy"`
	DecidedAt time.Time `json:"decidedAt"`
}

// DecideProposal records the human accept/reject of an agent proposal.
// Only the project owner or a workspace admin may decide; the decision
// row is upserted (latest state) and a timeline note makes the decision
// visible in the project history.
func (s *Store) DecideProposal(ctx context.Context, actorID, updateID uuid.UUID, decision, reason string) (*ProposalDecision, error) {
	if decision != "accepted" && decision != "rejected" {
		return nil, errors.New("decision must be accepted or rejected")
	}
	var out *ProposalDecision
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var projectID uuid.UUID
		var kind string
		var content string
		if err := tx.QueryRow(ctx,
			`SELECT project_id, kind, content FROM insideout.project_updates WHERE id = $1 FOR UPDATE`,
			updateID).Scan(&projectID, &kind, &content); err != nil {
			if err == pgx.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
		if kind != "agent_proposal" {
			return ErrForbidden
		}
		var workspaceID uuid.UUID
		var ownerID *uuid.UUID
		if err := tx.QueryRow(ctx,
			`SELECT workspace_id, owner_id FROM insideout.projects WHERE id = $1`, projectID,
		).Scan(&workspaceID, &ownerID); err != nil {
			return err
		}
		if ownerID == nil || *ownerID != actorID {
			admin, err := isWorkspaceAdmin(ctx, tx, workspaceID, actorID)
			if err != nil {
				return err
			}
			if !admin {
				return ErrForbidden
			}
		}
		var d ProposalDecision
		if err := tx.QueryRow(ctx, `
			INSERT INTO insideout.proposal_decisions (update_id, decision, reason, decided_by)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (update_id) DO UPDATE
			  SET decision = EXCLUDED.decision, reason = EXCLUDED.reason,
			      decided_by = EXCLUDED.decided_by, decided_at = now()
			RETURNING update_id, decision, reason, decided_by, decided_at`,
			updateID, decision, reason, actorID,
		).Scan(&d.UpdateID, &d.Decision, &d.Reason, &d.DecidedBy, &d.DecidedAt); err != nil {
			return err
		}
		// The decision is part of the project history, visible to everyone.
		if _, err := tx.Exec(ctx, `
			INSERT INTO insideout.project_updates (project_id, author_id, kind, content)
			VALUES ($1, $2, 'note', $3)`, projectID, actorID,
			"[proposal "+decision+"] "+firstLine(content)+noteSuffix(reason)); err != nil {
			return err
		}
		out = &d
		return nil
	})
	return out, err
}

func isWorkspaceAdmin(ctx context.Context, tx pgx.Tx, workspaceID, userID uuid.UUID) (bool, error) {
	var admin bool
	err := tx.QueryRow(ctx,
		`SELECT insideout._is_admin($1, $2)`, workspaceID, userID).Scan(&admin)
	return admin, err
}

func firstLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	return s
}

func noteSuffix(reason string) string {
	if reason == "" {
		return ""
	}
	return " — " + reason
}
