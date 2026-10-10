package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

// FEAT-04 / REL-07 (review 3): the loop's last leg. A human-entered
// observation lands as evidence of a new validation kind, and a done leaf
// becomes validated only through a human confirm carrying a reason — no
// generic status move may claim it.

// A typed observation is recorded as evidence of the validation kind,
// leaves only, by the project owner (a plain member may read, not write).
func TestValidationObservation_IsEvidenceOfTheValidationKind(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	member := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Validation WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, member.ID, ws.Code); err != nil {
		t.Fatalf("join: %v", err)
	}
	proj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Validation project", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	root, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, nil, "Ship", "")
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	leaf, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, &root.ID, "Tour", "")
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}

	observed := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	if err := st.AddValidationObservation(ctx, admin.ID, leaf.ID,
		"4 of 5 pilot users finished the tour without help", "pilot check-in thread", observed,
	); err != nil {
		t.Fatalf("add observation: %v", err)
	}
	rows, err := st.ListRoadmapEvidence(ctx, admin.ID, leaf.ID)
	if err != nil {
		t.Fatalf("list evidence: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d evidence rows, want 1", len(rows))
	}
	got := rows[0]
	if got.Kind != "validation" {
		t.Fatalf("Kind = %q, want %q", got.Kind, "validation")
	}
	if got.Detail != "4 of 5 pilot users finished the tour without help" {
		t.Fatalf("Detail = %q", got.Detail)
	}
	if got.SourceURL != "pilot check-in thread" {
		t.Fatalf("SourceURL = %q", got.SourceURL)
	}
	if got.ObservedAt == nil || got.ObservedAt.UTC().Format("2006-01-02") != "2026-10-02" {
		t.Fatalf("ObservedAt = %v, want 2026-10-02", got.ObservedAt)
	}

	// Observations attach to leaves only — a branch with children would
	// blur which work was observed.
	if err := st.AddValidationObservation(ctx, admin.ID, root.ID, "on the root", "", observed); !errors.Is(err, ErrConflict) {
		t.Fatalf("observation on a non-leaf: err = %v, want ErrConflict", err)
	}

	// The evidence write contract holds: owner or workspace admin, so a
	// plain member's observation is refused.
	if err := st.AddValidationObservation(ctx, member.ID, leaf.ID, "as a member", "", observed); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member observation: err = %v, want ErrForbidden", err)
	}
}

// Reaching validated takes a human confirm with a reason: the generic
// status move cannot enter the state, a blank reason is refused, and the
// milestone (who, when, why) rides the node through the read path.
func TestValidatedState_RequiresHumanConfirmWithReason(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	member := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Validated WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, member.ID, ws.Code); err != nil {
		t.Fatalf("join: %v", err)
	}
	proj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Validated project", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	root, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, nil, "Ship", "")
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	leaf, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, &root.ID, "Tour", "")
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}
	for _, status := range []string{"in_progress", "done"} {
		if _, err := st.UpdateRoadmapNode(ctx, admin.ID, leaf.ID, RoadmapNodeFields{Status: sptr(status)}); err != nil {
			t.Fatalf("walk to %s: %v", status, err)
		}
	}

	// The generic move cannot enter validated: it carries no reason, so it
	// is not a human confirm.
	if _, err := st.UpdateRoadmapNode(ctx, admin.ID, leaf.ID, RoadmapNodeFields{Status: sptr("validated")}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("generic move into validated: err = %v, want ErrInvalidTransition", err)
	}

	// The confirm must say why.
	if _, err := st.ValidateRoadmapNode(ctx, admin.ID, leaf.ID, "   "); !errors.Is(err, ErrValidation) {
		t.Fatalf("blank reason: err = %v, want ErrValidation", err)
	}

	// A plain member cannot confirm — the plan writer decides.
	if _, err := st.ValidateRoadmapNode(ctx, member.ID, leaf.ID, "activation held at 68%"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member confirm: err = %v, want ErrForbidden", err)
	}

	reason := "7-day retention rose from 40% to 68%"
	n, err := st.ValidateRoadmapNode(ctx, admin.ID, leaf.ID, reason)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if n.Status != "validated" {
		t.Fatalf("Status = %q, want validated", n.Status)
	}
	if n.ValidationReason != reason {
		t.Fatalf("ValidationReason = %q, want %q", n.ValidationReason, reason)
	}
	if n.ValidatedBy == nil || *n.ValidatedBy != admin.ID {
		t.Fatalf("ValidatedBy = %v, want %s", n.ValidatedBy, admin.ID)
	}
	if n.ValidatedAt == nil {
		t.Fatal("ValidatedAt = nil, want the confirm time")
	}

	nodes, err := st.ListRoadmap(ctx, admin.ID, proj.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var listed *RoadmapNode
	for i := range nodes {
		if nodes[i].ID == leaf.ID {
			listed = &nodes[i]
		}
	}
	if listed == nil || listed.Status != "validated" || listed.ValidationReason != reason {
		t.Fatalf("listed node = %+v, want validated with the reason", listed)
	}

	// The next version may reopen the node without erasing the milestone.
	reopened, err := st.UpdateRoadmapNode(ctx, admin.ID, leaf.ID, RoadmapNodeFields{Status: sptr("in_progress")})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if reopened.ValidationReason != reason {
		t.Fatalf("reopen dropped the reason: %q", reopened.ValidationReason)
	}

	// Only finished work validates: the fresh leaf is still pending.
	fresh, err := st.CreateRoadmapNode(ctx, admin.ID, proj.ID, &root.ID, "Second tour", "")
	if err != nil {
		t.Fatalf("create fresh leaf: %v", err)
	}
	if _, err := st.ValidateRoadmapNode(ctx, admin.ID, fresh.ID, "too early"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("pending validate: err = %v, want ErrInvalidTransition", err)
	}
}
