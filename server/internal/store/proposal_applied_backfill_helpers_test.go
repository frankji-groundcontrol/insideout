package store

import (
	"context"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/db"
	"github.com/google/uuid"
)

// rv4 whole-change review, cluster w4-rv-store-db, Root C (rv4-FR-F38, F25):
// proposal_items.applied_at was added with no backfill, so what the old build
// had applied read as not applied. These helpers serve
// proposal_apply_once_test.go.

// rvStoreDbBackfillFile is the upgrade step that stamps the items applied
// before applied_at existed.
const rvStoreDbBackfillFile = "migrations/20261018221000_proposal_items_applied_backfill.sql"

// rvStoreDbBackfill runs that step the way the migrate runner does: as the
// owner, over the whole database.
func rvStoreDbBackfill(t *testing.T, owner *Store) {
	t.Helper()
	text, err := db.Migrations.ReadFile(rvStoreDbBackfillFile)
	if err != nil {
		t.Fatalf("no upgrade step stamps the items applied before applied_at existed: %v", err)
	}
	if _, err := owner.Pool.Exec(context.Background(), string(text)); err != nil {
		t.Fatalf("run %s: %v", rvStoreDbBackfillFile, err)
	}
}

// rvStoreDbAppliedAt is applied_at ("" while NULL) of every item of the given
// proposals, keyed by item id, read as the owner.
func rvStoreDbAppliedAt(t *testing.T, owner *Store, updateIDs ...uuid.UUID) map[uuid.UUID]string {
	t.Helper()
	out := map[uuid.UUID]string{}
	for _, id := range updateIDs {
		rows, err := owner.Pool.Query(context.Background(),
			`SELECT id, COALESCE(applied_at::text, '') FROM insideout.proposal_items WHERE update_id = $1`, id)
		if err != nil {
			t.Fatalf("read applied_at: %v", err)
		}
		for rows.Next() {
			var item uuid.UUID
			var at string
			if err := rows.Scan(&item, &at); err != nil {
				rows.Close()
				t.Fatalf("scan applied_at: %v", err)
			}
			out[item] = at
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			t.Fatalf("read applied_at: %v", err)
		}
	}
	return out
}

// rvStoreDbStamped counts the stamped items of one proposal.
func rvStoreDbStamped(t *testing.T, owner *Store, updateID uuid.UUID) (stamped, total int) {
	t.Helper()
	for _, at := range rvStoreDbAppliedAt(t, owner, updateID) {
		total++
		if at != "" {
			stamped++
		}
	}
	return stamped, total
}

// rvStoreDbAnotherProposal files one more agent proposal, with one add_node
// item, in the project of an applyFixture.
func rvStoreDbAnotherProposal(t *testing.T, st *Store, f *applyFixture, title string) *applyFixture {
	t.Helper()
	ctx := context.Background()
	upd, err := st.AddProjectUpdate(ctx, f.owner.ID, f.proj.ID, "agent_proposal", "[agent proposal/structure] "+title)
	if err != nil {
		t.Fatalf("add proposal: %v", err)
	}
	if err := st.AddProposalItems(ctx, f.owner.ID, upd.ID, []ProposalItem{{Action: "add_node", Title: title}}); err != nil {
		t.Fatalf("add item: %v", err)
	}
	return &applyFixture{owner: f.owner, ws: f.ws, proj: f.proj, upd: upd}
}
