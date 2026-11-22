package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/frankji-groundcontrol/insideout/server/internal/roadmaptime"
)

// What a person sees at the first glance of a root they are in charge of
// (2026-11-22 plan, S2): the work below it, what it waits on outside, and
// what outside waits on it. Only unfinished work waits or blocks.

const unfinished = `NOT IN ('done', 'validated')`

// subtreesOf is the recursive CTE "sub": (root, id) for each of the roots in
// $1 and every node below it. Prefix it with WITH RECURSIVE.
const subtreesOf = `sub AS (
		SELECT id AS root, id FROM insideout.roadmap_nodes WHERE id = ANY($1)
		UNION ALL
		SELECT sub.root, n.id FROM insideout.roadmap_nodes n JOIN sub ON n.parent_id = sub.id
	)`

// firstGlanceTx fills Subtree, WaitsOn and Blocking on each root.
func firstGlanceTx(ctx context.Context, tx pgx.Tx, roots []MyRoadmapRoot) error {
	if len(roots) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(roots))
	at := make(map[uuid.UUID]*MyRoadmapRoot, len(roots))
	for i := range roots {
		ids[i], at[roots[i].Node.ID] = roots[i].Node.ID, &roots[i]
	}
	if err := subtreesTx(ctx, tx, ids, at); err != nil {
		return err
	}
	return crossLinksTx(ctx, tx, ids, at)
}

// subtreesTx reads every node below the roots, earliest deadline first, each
// with the unfinished nodes it waits on.
func subtreesTx(ctx context.Context, tx pgx.Tx, ids []uuid.UUID, at map[uuid.UUID]*MyRoadmapRoot) error {
	rows, err := tx.Query(ctx, `WITH RECURSIVE `+subtreesOf+`
		SELECT sub.root, n.id, n.title, n.status, n.deadline, n.acceptance, COALESCE(u.username, ''),
		       NOT EXISTS (SELECT 1 FROM insideout.roadmap_nodes c WHERE c.parent_id = n.id)
		FROM sub JOIN insideout.roadmap_nodes n ON n.id = sub.id
		LEFT JOIN insideout.users u ON u.id = n.owner_user_id
		ORDER BY sub.root, n.deadline NULLS LAST, n.created_at, n.id`, ids)
	if err != nil {
		return err
	}
	var all []uuid.UUID
	for rows.Next() {
		var root, id uuid.UUID
		var n roadmaptime.Node
		if err := rows.Scan(&root, &id, &n.Title, &n.Status, &n.Deadline, &n.Acceptance, &n.OwnerName, &n.Leaf); err != nil {
			rows.Close()
			return err
		}
		n.ID, all = id.String(), append(all, id)
		at[root].Subtree = append(at[root].Subtree, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	waits, err := waitsTx(ctx, tx, `d.node_id = ANY($1)`, all)
	if err != nil {
		return err
	}
	for _, r := range at {
		for i := range r.Subtree {
			r.Subtree[i].WaitsOn = waits[r.Subtree[i].ID]
		}
	}
	return nil
}

// crossLinksTx reads the links with one unfinished end inside a root and the
// other unfinished end outside it.
func crossLinksTx(ctx context.Context, tx pgx.Tx, ids []uuid.UUID, at map[uuid.UUID]*MyRoadmapRoot) error {
	rows, err := tx.Query(ctx, `WITH RECURSIVE `+subtreesOf+`,
		links AS (
			SELECT sub.root, true AS waits, d.depends_on_id AS other, d.node_id AS inner_id
			FROM sub JOIN insideout.roadmap_dependencies d ON d.node_id = sub.id
			UNION ALL
			SELECT sub.root, false, d.node_id, d.depends_on_id
			FROM sub JOIN insideout.roadmap_dependencies d ON d.depends_on_id = sub.id
		)
		SELECT DISTINCT l.root, l.waits, o.id, o.title, o.status, COALESCE(u.username, '')
		FROM links l
		JOIN insideout.roadmap_nodes o ON o.id = l.other
		JOIN insideout.roadmap_nodes i ON i.id = l.inner_id
		LEFT JOIN insideout.users u ON u.id = o.owner_user_id
		WHERE o.status `+unfinished+` AND i.status `+unfinished+`
		  AND NOT EXISTS (SELECT 1 FROM sub s2 WHERE s2.root = l.root AND s2.id = o.id)
		ORDER BY l.root, l.waits, o.title, o.id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var root uuid.UUID
		var waits bool
		var l RoadmapLink
		if err := rows.Scan(&root, &waits, &l.ID, &l.Title, &l.Status, &l.OwnerName); err != nil {
			return err
		}
		if waits {
			at[root].WaitsOn = append(at[root].WaitsOn, l)
		} else {
			at[root].Blocking = append(at[root].Blocking, l)
		}
	}
	return rows.Err()
}

// waitsTx maps each waiting node d.node_id that matches where ($1 is arg) to
// the unfinished nodes it waits on, oldest link first; a node that waits on
// nothing unfinished is absent.
func waitsTx(ctx context.Context, tx pgx.Tx, where string, arg any) (map[string][]roadmaptime.Ref, error) {
	rows, err := tx.Query(ctx, `
		SELECT d.node_id, t.id, t.title FROM insideout.roadmap_dependencies d
		JOIN insideout.roadmap_nodes t ON t.id = d.depends_on_id
		WHERE `+where+` AND t.status `+unfinished+`
		ORDER BY d.node_id, d.created_at, d.depends_on_id`, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]roadmaptime.Ref{}
	for rows.Next() {
		var node, id uuid.UUID
		var title string
		if err := rows.Scan(&node, &id, &title); err != nil {
			return nil, err
		}
		out[node.String()] = append(out[node.String()], roadmaptime.Ref{ID: id.String(), Title: title})
	}
	return out, rows.Err()
}

// RoadmapWaits maps each node of the project to the unfinished nodes it waits
// on, for Progress to rank Next. The caller has read the project's nodes, so
// membership is checked; RLS keeps the read inside it.
func (s *Store) RoadmapWaits(ctx context.Context, actorID, projectID uuid.UUID) (map[string][]roadmaptime.Ref, error) {
	var out map[string][]roadmaptime.Ref
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		var err error
		out, err = waitsTx(ctx, tx, `t.project_id = $1`, projectID)
		return err
	})
	return out, err
}
