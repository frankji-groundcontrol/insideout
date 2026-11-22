package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RoadmapGlance is one node opened as its person in charge sees it
// (2026-11-22 view-as plan): the node with the first glance a `mine` root
// carries, and who is in charge of it.
type RoadmapGlance struct {
	MyRoadmapRoot
	// InCharge is nil only when the node, its ancestors and the project
	// have no owner.
	InCharge *RoadmapInCharge
}

// RoadmapInCharge is the person in charge of a node: its owner; else the
// nearest owned ancestor's owner, Via naming that ancestor; else the project
// owner, with ProjectOwner set.
type RoadmapInCharge struct {
	UserID       uuid.UUID
	Name         string
	Via          *RoadmapNodeRef
	ProjectOwner bool
}

// ReadRoadmapGlance opens one node for a current workspace member with its
// first glance and its person in charge. A node the actor cannot see answers
// ErrNotFound (RLS) or ErrForbidden (not a member).
func (s *Store) ReadRoadmapGlance(ctx context.Context, actorID, nodeID uuid.UUID) (*RoadmapGlance, error) {
	out := &RoadmapGlance{}
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
		roots := []MyRoadmapRoot{{Node: *node, WorkspaceID: wsID}}
		if err := firstGlanceTx(ctx, tx, roots); err != nil {
			return err
		}
		out.MyRoadmapRoot = roots[0]
		paths, err := ancestorsTx(ctx, tx, []uuid.UUID{nodeID})
		if err != nil {
			return err
		}
		out.Ancestors = paths[nodeID]
		out.InCharge, err = inChargeTx(ctx, tx, *node, out.Ancestors)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// inChargeTx resolves the person in charge of n, whose ancestors run from the
// root down to its parent.
func inChargeTx(ctx context.Context, tx pgx.Tx, n RoadmapNode, ancestors []RoadmapNodeRef) (*RoadmapInCharge, error) {
	if n.OwnerID != nil {
		return &RoadmapInCharge{UserID: *n.OwnerID, Name: n.OwnerName}, nil
	}
	c := &RoadmapInCharge{}
	for i := len(ancestors) - 1; i >= 0 && c.Via == nil; i-- {
		if a := ancestors[i]; a.OwnerID != nil {
			c.Via = &a
		}
	}
	// One read for either the ancestor's owner or the project owner.
	err := tx.QueryRow(ctx, `
		SELECT u.id, COALESCE(u.username, '') FROM insideout.users u
		WHERE u.id = COALESCE($1, (SELECT owner_id FROM insideout.projects WHERE id = $2))`,
		viaOwner(c.Via), n.ProjectID).Scan(&c.UserID, &c.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	c.ProjectOwner = c.Via == nil
	return c, err
}

func viaOwner(via *RoadmapNodeRef) *uuid.UUID {
	if via == nil {
		return nil
	}
	return via.OwnerID
}
