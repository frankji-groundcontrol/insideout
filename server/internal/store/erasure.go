package store

import (
	"context"

	"github.com/google/uuid"
)

// EraseUser will tombstone one account. This revision does not.
func (s *Store) EraseUser(ctx context.Context, userID uuid.UUID) error {
	return nil
}
