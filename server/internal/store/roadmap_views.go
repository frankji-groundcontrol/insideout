package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/frankji-groundcontrol/insideout/server/internal/roadmaptime"
)

// Node views as a reader gets them: display names, the actor's CanWrite and
// the roll-up over the leaves below (rv5-TREEB-03, TREEB-08). A list read
// computes roll-ups from the list itself; any other read fetches the nodes
// below with one query, however many nodes it returns.

// setRollups fills Rollup on every node of a list that holds each node
// together with everything below it.
func setRollups(nodes []RoadmapNode) {
	tree := make([]roadmaptime.TreeNode, len(nodes))
	for i, n := range nodes {
		tree[i] = treeNode(n.ID, n.ParentID, n.Status, n.Deadline)
	}
	r := roadmaptime.Rollups(tree)
	for i := range nodes {
		nodes[i].Rollup = r[nodes[i].ID.String()]
	}
}

func treeNode(id uuid.UUID, parentID *uuid.UUID, status string, deadline *time.Time) roadmaptime.TreeNode {
	t := roadmaptime.TreeNode{ID: id.String(), Status: status, Deadline: deadline}
	if parentID != nil {
		t.ParentID = parentID.String()
	}
	return t
}

// readNodeViewsTx reads the given nodes, in the given order, as the actor sees
// them inside the caller's transaction, each with its roll-up. After a write
// CanWrite is computed, not assumed: handing back the node you were handed
// leaves you without the right to write it. ErrNotFound when one is missing.
func readNodeViewsTx(ctx context.Context, tx pgx.Tx, actorID uuid.UUID, ids []uuid.UUID) ([]RoadmapNode, error) {
	rows, err := tx.Query(ctx, nodeViewSelect()+` WHERE n.id = ANY($2)`, actorID, ids)
	if err != nil {
		return nil, err
	}
	byID := map[uuid.UUID]RoadmapNode{}
	for rows.Next() {
		n, err := scanNodeView(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		byID[n.ID] = n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]RoadmapNode, len(ids))
	for i, id := range ids {
		n, ok := byID[id]
		if !ok {
			return nil, ErrNotFound
		}
		out[i] = n
	}
	return out, attachRollupsTx(ctx, tx, out)
}

// readNodeViewTx is readNodeViewsTx for one node.
func readNodeViewTx(ctx context.Context, tx pgx.Tx, actorID, nodeID uuid.UUID) (*RoadmapNode, error) {
	out, err := readNodeViewsTx(ctx, tx, actorID, []uuid.UUID{nodeID})
	if err != nil {
		return nil, err
	}
	return &out[0], nil
}

// attachRollupsTx sets Rollup on each node from the nodes below it, read in
// one recursive query.
func attachRollupsTx(ctx context.Context, tx pgx.Tx, nodes []RoadmapNode) error {
	if len(nodes) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	rows, err := tx.Query(ctx, `
		WITH RECURSIVE d AS (
			SELECT id, parent_id, status, deadline FROM insideout.roadmap_nodes WHERE id = ANY($1)
			UNION
			SELECT n.id, n.parent_id, n.status, n.deadline FROM insideout.roadmap_nodes n JOIN d ON n.parent_id = d.id
		)
		SELECT id, parent_id, status, deadline FROM d`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	var tree []roadmaptime.TreeNode
	for rows.Next() {
		var n RoadmapNode
		if err := rows.Scan(&n.ID, &n.ParentID, &n.Status, &n.Deadline); err != nil {
			return err
		}
		tree = append(tree, treeNode(n.ID, n.ParentID, n.Status, n.Deadline))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	r := roadmaptime.Rollups(tree)
	for i := range nodes {
		nodes[i].Rollup = r[nodes[i].ID.String()]
	}
	return nil
}
