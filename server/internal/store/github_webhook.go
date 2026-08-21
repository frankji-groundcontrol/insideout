package store

import (
	"context"

	"github.com/google/uuid"
)

// ProjectOwner pairs a project with an owner used by system-triggered
// flows (the GitHub webhook) to run as, since deliveries carry no user
// identity. Resolved by the DEFINER helper _projects_by_repo; all
// follow-up reads/writes go through the normal user-scoped store
// methods with that owner id, so RLS still governs everything else.
type ProjectOwner struct {
	ProjectID uuid.UUID
	OwnerID   uuid.UUID
}

// ProjectsByRepo resolves every project linked to a GitHub repository
// URL. Runs through the SECURITY DEFINER helper because the webhook has
// no RLS identity of its own.
func (s *Store) ProjectsByRepo(ctx context.Context, repoURL string) ([]ProjectOwner, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT project_id, owner_id FROM insideout._projects_by_repo($1)`, repoURL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProjectOwner
	for rows.Next() {
		var po ProjectOwner
		if err := rows.Scan(&po.ProjectID, &po.OwnerID); err != nil {
			return nil, err
		}
		out = append(out, po)
	}
	return out, rows.Err()
}
