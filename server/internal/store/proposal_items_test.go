package store

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Applying a proposal must create its nodes in the order they were
// proposed, not in whatever order a created_at tie happens to return
// (BUG-019).
func TestProposal_ApplyKeepsItemOrder(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	admin := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Proposal WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, admin.ID, ws.ID, "Proposal", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	upd, err := st.AddProjectUpdate(ctx, admin.ID, proj.ID, "agent_proposal", "[agent proposal/structure] four steps")
	if err != nil {
		t.Fatalf("add proposal: %v", err)
	}
	titles := []string{"Intake", "Verification", "Handoff", "Archive"}
	items := make([]ProposalItem, len(titles))
	for i, title := range titles {
		items[i] = ProposalItem{UpdateID: upd.ID, Action: "add_node", Title: title}
	}
	if err := st.AddProposalItems(ctx, admin.ID, upd.ID, items); err != nil {
		t.Fatalf("add items: %v", err)
	}

	listed, err := st.ListProposalItems(ctx, admin.ID, upd.ID)
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	for i, title := range titles {
		if i >= len(listed) || listed[i].Title != title {
			got := make([]string, len(listed))
			for j, it := range listed {
				got[j] = it.Title
			}
			t.Fatalf("listed order: got %v, want %v", got, titles)
		}
	}

	if _, err := st.ApplyProposalItems(ctx, admin.ID, proj.ID, upd.ID); err != nil {
		t.Fatalf("apply: %v", err)
	}
	nodes, err := st.ListRoadmap(ctx, admin.ID, proj.ID)
	if err != nil {
		t.Fatalf("list roadmap: %v", err)
	}
	got := make([]string, len(nodes))
	for i, n := range nodes {
		got[i] = n.Title
	}
	if len(got) != len(titles) {
		t.Fatalf("nodes: got %v, want %v", got, titles)
	}
	for i := range titles {
		if got[i] != titles[i] {
			t.Fatalf("applied order: got %v, want %v", got, titles)
		}
	}
}

// A plain member may read the roadmap but must not change it, and must
// not file a proposal. The owner still can (BUG-020).
func TestRoadmap_MemberCannotWrite(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	owner := mkUser(t, st)
	reader := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, owner.ID, "Reader WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, reader.ID, ws.Code); err != nil {
		t.Fatalf("join: %v", err)
	}
	proj, err := st.CreateProject(ctx, owner.ID, ws.ID, "Reader", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	node, err := st.CreateRoadmapNode(ctx, owner.ID, proj.ID, nil, "Only the owner", "")
	if err != nil {
		t.Fatalf("owner create: %v", err)
	}

	if _, err := st.ListRoadmap(ctx, reader.ID, proj.ID); err != nil {
		t.Fatalf("member should still read: %v", err)
	}
	if _, err := st.CreateRoadmapNode(ctx, reader.ID, proj.ID, nil, "A reader must not add this", ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member create: want ErrForbidden, got %v", err)
	}
	if _, err := st.UpdateRoadmapNode(ctx, reader.ID, node.ID, RoadmapNodeFields{Status: sp("done")}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member update: want ErrForbidden, got %v", err)
	}
	if _, err := st.MoveRoadmapNode(ctx, reader.ID, node.ID, nil, 0); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member move: want ErrForbidden, got %v", err)
	}
	if err := st.DeleteRoadmapNode(ctx, reader.ID, node.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member delete: want ErrForbidden, got %v", err)
	}
	if _, err := st.AddProjectUpdate(ctx, reader.ID, proj.ID, "agent_proposal", "a reader proposes"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member proposal: want ErrForbidden, got %v", err)
	}
	if _, err := st.AddProjectUpdate(ctx, reader.ID, proj.ID, "note", "a reader notes"); err != nil {
		t.Fatalf("member note should still be allowed: %v", err)
	}

	nodes, err := st.ListRoadmap(ctx, owner.ID, proj.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Title != "Only the owner" {
		t.Fatalf("tree changed by a member: %+v", nodes)
	}
	// keep the import honest if a future assertion needs it
	_ = uuid.Nil
}

// rv4-FEAT-02, the migration's two guards (green-only: the column did not exist
// at the base). applied_at is the one thing a proposal item lets the app role
// change, and only the people who may apply the proposal may change it.
func TestProposalItems_OnlyThePlanWriterStampsAppliedAtAndNothingElseChanges(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	f := newApplyFixture(t, st, ProposalItem{Title: "Intake"})
	member := mkUser(t, st)
	if _, err := st.JoinWorkspace(ctx, member.ID, f.ws.Code); err != nil {
		t.Fatalf("join: %v", err)
	}
	items, err := st.ListProposalItems(ctx, f.owner.ID, f.upd.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("list items: %d, %v", len(items), err)
	}
	itemID := items[0].ID
	run := func(actor uuid.UUID, sql string) (int64, error) {
		var n int64
		err := st.withUserContext(ctx, actor, func(tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, sql, itemID)
			n = tag.RowsAffected()
			return err
		})
		return n, err
	}

	// Another column is not writable, whoever asks (table-level UPDATE was revoked).
	var pgErr *pgconn.PgError
	if _, err := run(f.owner.ID, `UPDATE insideout.proposal_items SET title = 'rewritten' WHERE id = $1`); !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Errorf("owner rewriting an item title: want permission denied (42501), got %v", err)
	}
	// A member who may not apply sees the row but updates none of it.
	if n, err := run(member.ID, `UPDATE insideout.proposal_items SET applied_at = now() WHERE id = $1`); err != nil || n != 0 {
		t.Errorf("member stamping applied_at: want 0 rows and no error, got %d, %v", n, err)
	}
	// The owner may.
	if n, err := run(f.owner.ID, `UPDATE insideout.proposal_items SET applied_at = now() WHERE id = $1 AND applied_at IS NULL`); err != nil || n != 1 {
		t.Errorf("owner stamping applied_at: want 1 row, got %d, %v", n, err)
	}
}

// A person who may not apply must be refused, not told the proposal is
// already applied: the claim is an RLS-filtered UPDATE that matches no row for
// them, which would otherwise read as "applied by someone else".
func TestProposalApply_AMemberWhoMayNotApplyIsRefused(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	f := newApplyFixture(t, st, ProposalItem{Title: "Intake"}, ProposalItem{Title: "Handoff"})
	member := mkUser(t, st)
	if _, err := st.JoinWorkspace(ctx, member.ID, f.ws.Code); err != nil {
		t.Fatalf("join: %v", err)
	}

	if ids, err := st.ApplyProposalItemsForUpdate(ctx, member.ID, f.upd.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member apply: want ErrForbidden, got %v (created %d)", err, len(ids))
	}
	f.wantTitles(t, st)
	ids, err := st.ApplyProposalItemsForUpdate(ctx, f.owner.ID, f.upd.ID)
	if err != nil || len(ids) != 2 {
		t.Fatalf("owner apply after the refused attempt: %d nodes, %v, want both items (the refusal must not claim them)", len(ids), err)
	}
}

// rv4-LEAD-PROPOSE-VALIDATE, the other half (green-only: AddProposal did not
// exist at the base). The proposal and its items are one transaction, so an
// item the database refuses leaves no proposal behind: a human could accept
// that one and apply nothing.
func TestAddProposal_AnItemTheDatabaseRefusesLeavesNoProposal(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	poison := "poison-" + uuid.NewString()
	lift := poisonTitleOn(t, "proposal_items", poison)
	f := newApplyFixture(t, st) // files one item-less proposal: the baseline
	proposals := func() int {
		updates, err := st.ListProjectUpdates(ctx, f.owner.ID, f.proj.ID, 100, nil)
		if err != nil {
			t.Fatalf("list updates: %v", err)
		}
		n := 0
		for _, u := range updates {
			if u.Kind == "agent_proposal" {
				n++
			}
		}
		return n
	}
	items := []ProposalItem{{Action: "add_node", Title: "Intake"}, {Action: "add_node", Title: poison}}

	if _, err := st.AddProposal(ctx, f.owner.ID, f.proj.ID, "[agent proposal/structure] refused", items); err == nil {
		t.Fatalf("AddProposal returned no error although the database refuses item 2")
	}
	if n := proposals(); n != 1 {
		t.Fatalf("the failed AddProposal left %d proposals, want only the baseline one", n)
	}

	lift()
	u, err := st.AddProposal(ctx, f.owner.ID, f.proj.ID, "[agent proposal/structure] fine", items)
	if err != nil {
		t.Fatalf("AddProposal once the cause is gone: %v", err)
	}
	if n := proposals(); n != 2 {
		t.Fatalf("after the good AddProposal there are %d proposals, want 2", n)
	}
	listed, err := st.ListProposalItems(ctx, f.owner.ID, u.ID)
	if err != nil || len(listed) != 2 || listed[0].Title != "Intake" || listed[1].Title != poison {
		t.Fatalf("items of the filed proposal: %+v, %v, want Intake then the formerly refused title", listed, err)
	}
}
