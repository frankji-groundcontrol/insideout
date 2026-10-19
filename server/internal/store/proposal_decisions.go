package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ProposalDecision is the human decision on an agent proposal — the
// Decision Log entry. The row holds the latest decision: deciding again
// overwrites it (a decision may be reversed). History is the project
// timeline, where every change of decision appends a "[proposal accepted]"
// or "[proposal rejected]" note (see DecideProposal); deciding what is
// already decided, with the same reason, is not a change.
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
// visible in the project history. Repeating the decision on record, with
// the same reason (a double tap, a retry, a second tab), writes neither
// and returns the record as it stands (see recordDecisionTx).
func (s *Store) DecideProposal(ctx context.Context, actorID, updateID uuid.UUID, decision, reason string) (*ProposalDecision, error) {
	d, _, err := s.decideProposal(ctx, actorID, updateID, decision, reason, false)
	return d, err
}

// DecideAndApplyProposal is DecideProposal that, for an acceptance with apply
// set, also applies the proposal's items (ApplyProposalItems) in the same
// transaction: the decision and the nodes it created land together or not at
// all, so a failed apply leaves the proposal open to accept again instead of
// decided with nothing built. Rejections ignore apply. Returns the ids of the
// nodes this call created: none when every item was already applied.
func (s *Store) DecideAndApplyProposal(ctx context.Context, actorID, updateID uuid.UUID, decision, reason string, apply bool) (*ProposalDecision, []uuid.UUID, error) {
	return s.decideProposal(ctx, actorID, updateID, decision, reason, apply && decision == "accepted")
}

func (s *Store) decideProposal(ctx context.Context, actorID, updateID uuid.UUID, decision, reason string, apply bool) (*ProposalDecision, []uuid.UUID, error) {
	if decision != "accepted" && decision != "rejected" {
		return nil, nil, errors.New("decision must be accepted or rejected")
	}
	var out *ProposalDecision
	var created []uuid.UUID
	var projectID uuid.UUID
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
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
		d, err := recordDecisionTx(ctx, tx, projectID, actorID, updateID, decision, reason, content)
		if err != nil {
			return err
		}
		if apply {
			if created, err = applyProposalItemsTx(ctx, tx, actorID, projectID, updateID); err != nil {
				return err
			}
		}
		out = d
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if len(created) > 0 {
		s.noteRoadmapChange(projectID)
	}
	return out, created, nil
}

// recordDecisionTx stores the decision on the proposal and its note on the
// project timeline, which is the history everyone reads. Deciding the same thing
// again with the same reason (a double tap, a retry, a second tab, the CLI after
// the web) changes nothing and returns the record on file: a repeat is not an
// event, and rewriting decided_at and decided_by would hide who decided and
// when. A different decision or a different reason is a change and writes both.
// The caller holds the proposal's row lock, so taps at once run in turn and the
// second finds the first's record. An accept with apply still applies whatever
// is left (the caller does that), which is how a proposal accepted without
// apply gets built later.
func recordDecisionTx(ctx context.Context, tx pgx.Tx, projectID, actorID, updateID uuid.UUID, decision, reason, content string) (*ProposalDecision, error) {
	var d ProposalDecision
	err := tx.QueryRow(ctx, `
		SELECT update_id, decision, reason, decided_by, decided_at
		FROM insideout.proposal_decisions WHERE update_id = $1`, updateID,
	).Scan(&d.UpdateID, &d.Decision, &d.Reason, &d.DecidedBy, &d.DecidedAt)
	if err == nil && d.Decision == decision && d.Reason == reason {
		return &d, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO insideout.proposal_decisions (update_id, decision, reason, decided_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (update_id) DO UPDATE
		  SET decision = EXCLUDED.decision, reason = EXCLUDED.reason,
		      decided_by = EXCLUDED.decided_by, decided_at = now()
		RETURNING update_id, decision, reason, decided_by, decided_at`,
		updateID, decision, reason, actorID,
	).Scan(&d.UpdateID, &d.Decision, &d.Reason, &d.DecidedBy, &d.DecidedAt); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO insideout.project_updates (project_id, author_id, kind, content)
		VALUES ($1, $2, 'note', $3)`, projectID, actorID,
		"[proposal "+decision+"] "+firstLine(content)+noteSuffix(reason)); err != nil {
		return nil, err
	}
	return &d, nil
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
