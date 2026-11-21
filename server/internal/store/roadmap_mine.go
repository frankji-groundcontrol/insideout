package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MyRoadmapRoot is one root of a person's work tree: a node they own whose
// ancestors they do not own, with its project and path.
type MyRoadmapRoot struct {
	Node        RoadmapNode
	ProjectName string
	Ancestors   []RoadmapNodeRef
}

// ListMyRoadmapRoots returns the actor's topmost owned nodes across the
// workspace's projects, ordered by project, then position (rv5-TREEB-04).
// Each person's tree is derived from the nodes they own (owner decision 1).
// ErrForbidden for a non-member.
func (s *Store) ListMyRoadmapRoots(ctx context.Context, actorID, workspaceID uuid.UUID) ([]MyRoadmapRoot, error) {
	out := []MyRoadmapRoot{}
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if _, err := requireMember(ctx, tx, workspaceID, actorID); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			WITH RECURSIVE owned AS (
				SELECT n.id, n.parent_id, n.position, n.created_at, p.id AS project_id, p.title AS project_title
				FROM insideout.roadmap_nodes n JOIN insideout.projects p ON p.id = n.project_id
				WHERE p.workspace_id = $1 AND n.owner_user_id = $2
			), up AS (
				SELECT o.id AS start, o.parent_id AS id FROM owned o WHERE o.parent_id IS NOT NULL
				UNION ALL
				SELECT up.start, a.parent_id FROM up JOIN insideout.roadmap_nodes a ON a.id = up.id
				WHERE a.parent_id IS NOT NULL
			)
			SELECT o.id, o.project_title FROM owned o
			WHERE NOT EXISTS (
				SELECT 1 FROM up JOIN insideout.roadmap_nodes a ON a.id = up.id
				WHERE up.start = o.id AND a.owner_user_id = $2
			)
			ORDER BY o.project_title, o.project_id, o.position, o.created_at`, workspaceID, actorID)
		if err != nil {
			return err
		}
		var ids []uuid.UUID
		var names []string
		for rows.Next() {
			var id uuid.UUID
			var name string
			if err := rows.Scan(&id, &name); err != nil {
				rows.Close()
				return err
			}
			ids, names = append(ids, id), append(names, name)
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
			out = append(out, MyRoadmapRoot{Node: n, ProjectName: names[i], Ancestors: paths[n.ID]})
		}
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
