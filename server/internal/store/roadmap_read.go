package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// descendantsOf is the recursive CTE "descendants": the node whose id is the
// given placeholder and every node below it. Prefix it with WITH RECURSIVE.
func descendantsOf(param string) string {
	return `descendants AS (
		SELECT id, project_id FROM insideout.roadmap_nodes WHERE id = ` + param + `
		UNION
		SELECT n.id, n.project_id FROM insideout.roadmap_nodes n
		JOIN descendants d ON n.parent_id = d.id
	)`
}

// nodeViewSelect reads nodes as "n" with their display names and the actor's
// write right; $1 is the actor. LEFT JOIN (D6): a removed author (ON DELETE
// SET NULL) must not drop the node; NULL names flow to the UI as "unknown".
// qualifyColumns keeps the join's shared id/created_at unambiguous. CanWrite
// calls insideout._can_write_node per row inside the one query.
func nodeViewSelect() string {
	return `SELECT ` + qualifyColumns("n", roadmapNodeColumns) + `,
		       u_cr.username, u_ed.username, COALESCE(u_ow.username, ''),
		       insideout._can_write_node(n.id, $1)
		FROM insideout.roadmap_nodes n
		LEFT JOIN insideout.users u_cr ON u_cr.id = n.created_by
		LEFT JOIN insideout.users u_ed ON u_ed.id = n.updated_by
		LEFT JOIN insideout.users u_ow ON u_ow.id = n.owner_user_id`
}

func scanNodeView(row pgx.Row) (RoadmapNode, error) {
	var n RoadmapNode
	err := row.Scan(&n.ID, &n.ProjectID, &n.ParentID, &n.Title, &n.Description, &n.Status, &n.Position, &n.Deadline, &n.CreatedAt, &n.UpdatedAt, &n.CreatedBy, &n.UpdatedBy, &n.Acceptance, &n.OwnerID, &n.ValidationReason, &n.ValidatedBy, &n.ValidatedAt, &n.CompletionReason, &n.CreatorName, &n.EditorName, &n.OwnerName, &n.CanWrite)
	if errors.Is(err, pgx.ErrNoRows) {
		return n, ErrNotFound
	}
	return n, err
}

// ListRoadmap returns every node of a project, roots first then grouped by
// parent, ordered by position — the frontend assembles the tree from this
// flat slice. Any workspace member may read.
func (s *Store) ListRoadmap(ctx context.Context, actorID, projectID uuid.UUID) ([]RoadmapNode, error) {
	return s.listRoadmap(ctx, actorID, projectID, nil)
}

// ListRoadmapSubtree is ListRoadmap restricted to rootID and the nodes below
// it, the root first. ErrNotFound when rootID is not a node of the project.
func (s *Store) ListRoadmapSubtree(ctx context.Context, actorID, projectID, rootID uuid.UUID) ([]RoadmapNode, error) {
	return s.listRoadmap(ctx, actorID, projectID, &rootID)
}

func (s *Store) listRoadmap(ctx context.Context, actorID, projectID uuid.UUID, rootID *uuid.UUID) ([]RoadmapNode, error) {
	var out []RoadmapNode
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		wsID, err := projectWorkspace(ctx, tx, projectID)
		if err != nil {
			return err
		}
		if _, err := requireMember(ctx, tx, wsID, actorID); err != nil {
			return err
		}
		if rootID != nil {
			var inProject bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM insideout.roadmap_nodes WHERE id = $1 AND project_id = $2)`,
				*rootID, projectID).Scan(&inProject); err != nil {
				return err
			}
			if !inProject {
				return ErrNotFound
			}
		}

		rows, err := tx.Query(ctx, `WITH RECURSIVE `+descendantsOf("$3")+`
			`+nodeViewSelect()+`
			WHERE n.project_id = $2 AND ($3::uuid IS NULL OR n.id IN (SELECT id FROM descendants))
			ORDER BY (n.id IS NOT DISTINCT FROM $3::uuid) DESC, n.parent_id NULLS FIRST, n.position, n.created_at`,
			actorID, projectID, rootID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			n, err := scanNodeView(rows)
			if err != nil {
				return err
			}
			out = append(out, n)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		setRollups(out)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetRoadmapNode returns one node for a workspace member, with its display
// names and the actor's CanWrite.
func (s *Store) GetRoadmapNode(ctx context.Context, actorID, nodeID uuid.UUID) (*RoadmapNode, error) {
	var n *RoadmapNode
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if _, err := nodeWorkspace(ctx, tx, nodeID); err != nil {
			return err
		}
		var err error
		n, err = readNodeViewTx(ctx, tx, actorID, nodeID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return n, nil
}

// NodeInSubtree reports whether nodeID is rootID or lies below it, as the
// actor sees the tree (an invisible node is in no subtree). It is the check
// behind a root-scoped agent token (rv5-TREEB-05).
func (s *Store) NodeInSubtree(ctx context.Context, actorID, rootID, nodeID uuid.UUID) (bool, error) {
	var in bool
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			WITH RECURSIVE up AS (
				SELECT id, parent_id FROM insideout.roadmap_nodes WHERE id = $2
				UNION
				SELECT n.id, n.parent_id FROM insideout.roadmap_nodes n JOIN up ON n.id = up.parent_id
			)
			SELECT EXISTS (SELECT 1 FROM up WHERE id = $1)`, rootID, nodeID).Scan(&in)
	})
	return in, err
}

// RoadmapNodeRef names a node in a path, with its owner (nil: none).
type RoadmapNodeRef struct {
	ID      uuid.UUID
	Title   string
	OwnerID *uuid.UUID
}

// RoadmapNodeRead is one node opened on its own: the node, its ancestors from
// the root down to its parent, and the project it belongs to.
type RoadmapNodeRead struct {
	Node        RoadmapNode
	Ancestors   []RoadmapNodeRef
	ProjectName string
}

// ReadRoadmapNode opens one node for a current workspace member. A node the
// actor cannot see answers ErrNotFound (RLS) or ErrForbidden (not a member).
func (s *Store) ReadRoadmapNode(ctx context.Context, actorID, nodeID uuid.UUID) (*RoadmapNodeRead, error) {
	out := &RoadmapNodeRead{Ancestors: []RoadmapNodeRef{}}
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		wsID, err := nodeWorkspace(ctx, tx, nodeID)
		if err != nil {
			return err
		}
		if _, err := requireMember(ctx, tx, wsID, actorID); err != nil {
			return err
		}
		node, err := readNodeViewTx(ctx, tx, actorID, nodeID)
		if err != nil {
			return err
		}
		out.Node = *node
		if err := tx.QueryRow(ctx, `SELECT title FROM insideout.projects WHERE id = $1`, out.Node.ProjectID).Scan(&out.ProjectName); err != nil {
			return err
		}
		paths, err := ancestorsTx(ctx, tx, []uuid.UUID{nodeID})
		out.Ancestors = paths[nodeID]
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
