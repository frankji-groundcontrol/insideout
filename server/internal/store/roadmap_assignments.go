package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Delegation on the record (rv5-TREEB-01, TREEB-09, TREEX-08). owner_user_id
// is the node's current responsible person; every change of it appends one
// insideout.roadmap_node_assignments row and one project feed row naming the
// node, in the transaction that changes it.

// NodeAssignment is one row of a node's owner history. Names are resolved at
// read time (an erased account reads as its placeholder).
type NodeAssignment struct {
	ID                               uuid.UUID
	Kind                             string // delegate, handback or cleared
	FromUserID, ToUserID, AssignedBy *uuid.UUID
	FromName, ToName, AssignedByName *string
	Note                             string
	CreatedAt                        time.Time
}

// ErrOwnerNotMember is the ErrForbidden for a new owner who is not a current
// member of the node's workspace.
var ErrOwnerNotMember = fmt.Errorf("%w: a node's owner must be a member of its workspace", ErrForbidden)

// feedKindFor maps a history kind to the project feed kind it writes.
var feedKindFor = map[string]string{"delegate": "node_delegated", "handback": "node_handed_back", "cleared": "node_handed_back"}

// recordOwnerChangeTx appends the history row and the feed row for one owner
// change the caller has just written. The feed text names the node, not the
// people: who-to-whom is read from the history, so an erased account's name
// never sits in feed text.
func recordOwnerChangeTx(ctx context.Context, tx pgx.Tx, actorID, nodeID uuid.UUID, from, to *uuid.UUID, kind, note string) error {
	var projectID uuid.UUID
	var title string
	if err := tx.QueryRow(ctx, `SELECT project_id, title FROM insideout.roadmap_nodes WHERE id = $1`, nodeID).Scan(&projectID, &title); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO insideout.roadmap_node_assignments (node_id, from_user_id, to_user_id, assigned_by, kind, note)
		VALUES ($1, $2, $3, $4, $5, $6)`, nodeID, from, to, actorID, kind, note); err != nil {
		return err
	}
	content := fmt.Sprintf("Delegated %q", title)
	switch {
	case kind == "handback":
		content = fmt.Sprintf("Handed back %q", title)
	case kind == "cleared":
		content = fmt.Sprintf("%q has no owner now", title)
	}
	if note != "" {
		content += ": " + note
	}
	return addNodeUpdateTx(ctx, tx, actorID, projectID, nodeID, feedKindFor[kind], content)
}

// addNodeUpdateTx writes a feed row about one node. The caller has already
// passed the node's write rule; the project_updates insert policy still holds
// the author to a current member.
func addNodeUpdateTx(ctx context.Context, tx pgx.Tx, actorID, projectID, nodeID uuid.UUID, kind, content string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO insideout.project_updates (project_id, author_id, kind, content, node_id)
		VALUES ($1, $2, $3, $4, $5)`, projectID, actorID, kind, content, nodeID)
	return err
}

// setNodeOwnerTx writes a new owner and records the change.
func setNodeOwnerTx(ctx context.Context, tx pgx.Tx, actorID, nodeID uuid.UUID, from, to *uuid.UUID, kind, note string) error {
	if _, err := tx.Exec(ctx, `UPDATE insideout.roadmap_nodes SET owner_user_id = $2, updated_by = $3 WHERE id = $1`,
		nodeID, to, actorID); err != nil {
		return err
	}
	return recordOwnerChangeTx(ctx, tx, actorID, nodeID, from, to, kind, note)
}

// nodeOwnerTx reads a node's current owner; ErrNotFound when not visible.
func nodeOwnerTx(ctx context.Context, tx pgx.Tx, nodeID uuid.UUID) (*uuid.UUID, error) {
	var owner *uuid.UUID
	err := tx.QueryRow(ctx, `SELECT owner_user_id FROM insideout.roadmap_nodes WHERE id = $1`, nodeID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return owner, err
}

func sameUser(a, b *uuid.UUID) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

// DelegateRoadmapNode makes toUserID the node's owner. It takes the rule every
// owner change takes (requireNodeManager: a plan writer, or a writer of the
// node's parent, so a delegate passes on descendants but not the node they
// were handed); the new owner must be a current member (ErrOwnerNotMember).
// Delegating to the current owner records nothing.
func (s *Store) DelegateRoadmapNode(ctx context.Context, actorID, nodeID, toUserID uuid.UUID, note string) (*RoadmapNode, error) {
	var n *RoadmapNode
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		if err := requireNodeManager(ctx, tx, nodeID, actorID); err != nil {
			return err
		}
		wsID, err := nodeWorkspace(ctx, tx, nodeID)
		if err != nil {
			return err
		}
		if _, err := requireMember(ctx, tx, wsID, toUserID); err != nil {
			return ErrOwnerNotMember
		}
		owner, err := nodeOwnerTx(ctx, tx, nodeID)
		if err != nil {
			return err
		}
		if !sameUser(owner, &toUserID) {
			if err := setNodeOwnerTx(ctx, tx, actorID, nodeID, owner, &toUserID, "delegate", note); err != nil {
				return err
			}
		}
		n, err = readNodeViewTx(ctx, tx, actorID, nodeID)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.noteRoadmapChange(n.ProjectID)
	return n, nil
}

// HandBackRoadmapNode returns a node from its current owner (only they may) to
// whoever made the latest 'delegate' row for it, or to nobody when that is
// unknown, the owner themself, or no longer a member.
func (s *Store) HandBackRoadmapNode(ctx context.Context, actorID, nodeID uuid.UUID, note string) (*RoadmapNode, error) {
	var n *RoadmapNode
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		owner, err := nodeOwnerTx(ctx, tx, nodeID)
		if err != nil {
			return err
		}
		if owner == nil || *owner != actorID {
			return ErrForbidden
		}
		var to *uuid.UUID
		err = tx.QueryRow(ctx, `
			SELECT a.assigned_by FROM insideout.roadmap_node_assignments a
			WHERE a.node_id = $1 AND a.kind = 'delegate'
			ORDER BY a.created_at DESC, a.id DESC LIMIT 1`, nodeID).Scan(&to)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if to != nil && *to == actorID {
			to = nil
		}
		if to != nil {
			wsID, err := nodeWorkspace(ctx, tx, nodeID)
			if err != nil {
				return err
			}
			if _, err := requireMember(ctx, tx, wsID, *to); err != nil {
				to = nil
			}
		}
		if err := setNodeOwnerTx(ctx, tx, actorID, nodeID, owner, to, "handback", note); err != nil {
			return err
		}
		n, err = readNodeViewTx(ctx, tx, actorID, nodeID)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.noteRoadmapChange(n.ProjectID)
	return n, nil
}

// ListNodeAssignments returns a node's owner history, newest first, to a
// current member of its workspace.
func (s *Store) ListNodeAssignments(ctx context.Context, actorID, nodeID uuid.UUID) ([]NodeAssignment, error) {
	out := []NodeAssignment{}
	err := s.withUserContext(ctx, actorID, func(tx pgx.Tx) error {
		wsID, err := nodeWorkspace(ctx, tx, nodeID)
		if err != nil {
			return err
		}
		if _, err := requireMember(ctx, tx, wsID, actorID); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			SELECT a.id, a.kind, a.from_user_id, a.to_user_id, a.assigned_by, uf.username, ut.username, ub.username, a.note, a.created_at
			FROM insideout.roadmap_node_assignments a
			LEFT JOIN insideout.users uf ON uf.id = a.from_user_id
			LEFT JOIN insideout.users ut ON ut.id = a.to_user_id
			LEFT JOIN insideout.users ub ON ub.id = a.assigned_by
			WHERE a.node_id = $1
			ORDER BY a.created_at DESC, a.id DESC`, nodeID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var a NodeAssignment
			if err := rows.Scan(&a.ID, &a.Kind, &a.FromUserID, &a.ToUserID, &a.AssignedBy, &a.FromName, &a.ToName, &a.AssignedByName, &a.Note, &a.CreatedAt); err != nil {
				return err
			}
			out = append(out, a)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// recordNodeUpdateTx puts a node edit on the record: an owner change in the
// history and the feed, and a node reaching done in the feed (rv5-TREEB-09).
func recordNodeUpdateTx(ctx context.Context, tx pgx.Tx, actorID uuid.UUID, n *RoadmapNode, fromStatus string, fromOwner *uuid.UUID) error {
	if !sameUser(fromOwner, n.OwnerID) {
		kind := "delegate"
		if n.OwnerID == nil {
			kind = "cleared"
		}
		if err := recordOwnerChangeTx(ctx, tx, actorID, n.ID, fromOwner, n.OwnerID, kind, ""); err != nil {
			return err
		}
	}
	if n.Status == "done" && fromStatus != "done" {
		return addNodeUpdateTx(ctx, tx, actorID, n.ProjectID, n.ID, "node_done", fmt.Sprintf("Finished %q", n.Title))
	}
	return nil
}

// clearOwnershipTx drops userID's ownership of every node in the workspace,
// on the record: a member who leaves stops owning its nodes (rv5-TREEB-12).
// The actor is whoever removes them (themself on a self-leave), called while
// the actor is still a member.
func clearOwnershipTx(ctx context.Context, tx pgx.Tx, actorID, workspaceID, userID uuid.UUID) error {
	rows, err := tx.Query(ctx, `
		SELECT n.id FROM insideout.roadmap_nodes n
		JOIN insideout.projects p ON p.id = n.project_id
		WHERE n.owner_user_id = $2 AND p.workspace_id = $1`, workspaceID, userID)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := setNodeOwnerTx(ctx, tx, actorID, id, &userID, nil, "cleared", ""); err != nil {
			return err
		}
	}
	return nil
}
