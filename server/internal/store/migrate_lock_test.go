package store

import (
	"context"
	"os"
	"testing"
	"time"
)

// REL-14: two processes booting into a database with pending migrations (a
// second replica, or a manual `insideout migrate` over a deploy boot) both read
// the same applied list and race on non-idempotent DDL; the loser exits. Migrate
// must wait for whoever holds the migration lock, then find nothing left to do.
func TestMigrate_WaitsForTheMigrationLock(t *testing.T) {
	ownerURL := os.Getenv("DATABASE_OWNER_URL")
	if ownerURL == "" {
		t.Skip("DATABASE_OWNER_URL not set, Migrate needs the owner role")
	}
	st := newTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := st.AttachMigratePool(ctx, ownerURL); err != nil {
		t.Fatalf("attach owner pool: %v", err)
	}

	// The other booter: a session-level lock on a connection of its own, pulled
	// out of the pool so that closing it ends the session and frees the lock.
	acquired, err := st.Pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	holder := acquired.Hijack()
	t.Cleanup(func() { holder.Close(context.Background()) })
	if _, err := holder.Exec(ctx, `SELECT pg_advisory_lock(hashtext('insideout:migrate'))`); err != nil {
		t.Fatalf("take the migration lock: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := st.Migrate(ctx)
		done <- err
	}()

	select {
	case err := <-done:
		t.Fatalf("Migrate returned (err=%v) while another process held the migration lock; it must wait", err)
	case <-time.After(500 * time.Millisecond):
	}

	holder.Close(ctx) // ends the session, which releases the lock
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Migrate after the lock was released: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Migrate still blocked 15s after the lock was released")
	}
}
