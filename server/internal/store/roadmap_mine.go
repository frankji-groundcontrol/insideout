package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/frankji-groundcontrol/insideout/server/internal/roadmaptime"
)

// MyRoadmapRoot is one root of a person's work tree: a node they are in
// charge of whose ancestors they are not, with its project, its path and what
// a first glance needs (2026-11-22 plan, S2).
type MyRoadmapRoot struct {
	Node        RoadmapNode
	WorkspaceID uuid.UUID
	ProjectName string
	Ancestors   []RoadmapNodeRef
	// Subtree is the root and every node below it, earliest deadline first,
	// reduced for roadmaptime.Assemble, which picks its Now items.
	Subtree []roadmaptime.Node
	// WaitsOn: unfinished nodes outside the root that unfinished work inside
	// waits on. Blocking: unfinished nodes outside that wait on unfinished
	// work inside. Both by title.
	WaitsOn, Blocking []RoadmapLink
}

// RoadmapLink is a node at the other end of a dependency, with its owner.
type RoadmapLink struct {
	ID        uuid.UUID
	Title     string
	Status    string
	OwnerName string
}

// inCharge is true for node n of project p when $2 owns n, or n is an unowned
// root and $2 is a plan writer of p: the project owner or a workspace admin
// (owner decision, 2026-11-22).
func inCharge(n, p string) string {
	return `(` + n + `.owner_user_id = $2 OR (` + n + `.parent_id IS NULL AND ` + n + `.owner_user_id IS NULL
		AND insideout._is_plan_writer(` + p + `, $2)))`
}

// ListMyRoadmapRoots returns the topmost nodes the actor is in charge of
// across the workspace's projects, ordered by project, then position
// (rv5-TREEB-04). Each person's tree is derived from the nodes they own
// (owner decision 1); an unowned root counts for its plan writers.
// ErrForbidden for a non-member.
func (s *Store) ListMyRoadmapRoots(ctx context.Context, actorID, workspaceID uuid.UUID) ([]MyRoadmapRoot, error) {
	return s.listMyRoadmapRoots(ctx, actorID, &workspaceID)
}

// ListAllMyRoadmapRoots is ListMyRoadmapRoots over every workspace the actor
// is a member of.
func (s *Store) ListAllMyRoadmapRoots(ctx context.Context, actorID uuid.UUID) ([]MyRoadmapRoot, error) {
	return s.listMyRoadmapRoots(ctx, actorID, nil)
}

func (s *Store) listMyRoadmapRoots(ctx context.Context, actorID uuid.UUID, workspaceID *uuid.UUID) ([]MyRoadmapRoot, error) {
	out := []MyRoadmapRoot{}
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if workspaceID != nil {
			if _, err := requireMember(ctx, tx, *workspaceID, actorID); err != nil {
				return err
			}
		}
		rows, err := tx.Query(ctx, `
			WITH RECURSIVE owned AS (
				SELECT n.id, n.parent_id, n.position, n.created_at, p.id AS project_id, p.title AS project_title, p.workspace_id
				FROM insideout.roadmap_nodes n JOIN insideout.projects p ON p.id = n.project_id
				WHERE ($1::uuid IS NULL OR p.workspace_id = $1) AND insideout._is_member(p.workspace_id, $2)
				  AND `+inCharge("n", "p.id")+`
			), up AS (
				SELECT o.id AS start, o.parent_id AS id FROM owned o WHERE o.parent_id IS NOT NULL
				UNION ALL
				SELECT up.start, a.parent_id FROM up JOIN insideout.roadmap_nodes a ON a.id = up.id
				WHERE a.parent_id IS NOT NULL
			)
			SELECT o.id, o.project_title, o.workspace_id FROM owned o
			WHERE NOT EXISTS (
				SELECT 1 FROM up JOIN insideout.roadmap_nodes a ON a.id = up.id
				WHERE up.start = o.id AND `+inCharge("a", "a.project_id")+`
			)
			ORDER BY o.project_title, o.project_id, o.position, o.created_at`, workspaceID, actorID)
		if err != nil {
			return err
		}
		var ids []uuid.UUID
		var roots []MyRoadmapRoot
		for rows.Next() {
			var id uuid.UUID
			var r MyRoadmapRoot
			if err := rows.Scan(&id, &r.ProjectName, &r.WorkspaceID); err != nil {
				rows.Close()
				return err
			}
			ids, roots = append(ids, id), append(roots, r)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		nodes, err := readNodeViewsTx(ctx, tx, actorID, ids)
		if err != nil {
			return err
		}
		paths, err := ancestorsTx(ctx, tx, ids)
		if err != nil {
			return err
		}
		for i, n := range nodes {
			roots[i].Node, roots[i].Ancestors = n, paths[n.ID]
		}
		if err := firstGlanceTx(ctx, tx, roots); err != nil {
			return err
		}
		out = append(out, roots...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ancestorsTx returns each node's ancestors from the root down to its parent,
// in one query; a root maps to an empty path.
func ancestorsTx(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) (map[uuid.UUID][]RoadmapNodeRef, error) {
	out := make(map[uuid.UUID][]RoadmapNodeRef, len(ids))
	for _, id := range ids {
		out[id] = []RoadmapNodeRef{}
	}
	rows, err := tx.Query(ctx, `
		WITH RECURSIVE up AS (
			SELECT n.id AS start, n.parent_id AS id, 1 AS depth FROM insideout.roadmap_nodes n
			WHERE n.id = ANY($1) AND n.parent_id IS NOT NULL
			UNION ALL
			SELECT up.start, a.parent_id, up.depth + 1 FROM up JOIN insideout.roadmap_nodes a ON a.id = up.id
			WHERE a.parent_id IS NOT NULL
		)
		SELECT up.start, a.id, a.title, a.owner_user_id FROM up JOIN insideout.roadmap_nodes a ON a.id = up.id
		ORDER BY up.start, up.depth DESC`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var start uuid.UUID
		var a RoadmapNodeRef
		if err := rows.Scan(&start, &a.ID, &a.Title, &a.OwnerID); err != nil {
			return nil, err
		}
		out[start] = append(out[start], a)
	}
	return out, rows.Err()
}
