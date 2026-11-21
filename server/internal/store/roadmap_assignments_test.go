package store

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// rv5 row 2.6 (TREEB-01): the owner history is append-only for the app role,
// and a row is written only in the actor's own name. Real PostgreSQL.
func TestTheOwnerHistoryIsAppendOnly(t *testing.T) {
	w := newDelegateWorld(t)
	ctx := context.Background()
	err := w.st.withUserContext(ctx, w.owner.ID, func(tx pgx.Tx) error {
		for _, q := range []string{
			`UPDATE insideout.roadmap_node_assignments SET note = 'rewritten' WHERE node_id = $1`,
			`DELETE FROM insideout.roadmap_node_assignments WHERE node_id = $1`,
		} {
			tag, err := tx.Exec(ctx, q, w.payments.ID)
			if err == nil && tag.RowsAffected() != 0 {
				t.Errorf("%s changed %d rows", q, tag.RowsAffected())
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("tx: %v", err)
	}
	err = w.st.withUserContext(ctx, w.owner.ID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO insideout.roadmap_node_assignments (node_id, assigned_by, kind) VALUES ($1, $2, 'cleared')`,
			w.payments.ID, w.member.ID)
		return err
	})
	if err == nil {
		t.Error("a row written in another user's name was accepted")
	}
	hist, err := w.st.ListNodeAssignments(ctx, w.other.ID, w.payments.ID)
	if err != nil || len(hist) != 1 || hist[0].Kind != "delegate" || hist[0].Note != "" {
		t.Errorf("history read by another member: %+v, %v; want the one untouched delegate row", hist, err)
	}
}
