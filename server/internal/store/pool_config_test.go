package store

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
)

// REL-11, REL-20: how many connections a replica may take from the shared
// pooler is stated in code and documented (docs/usage/deployment.md: 10 for
// requests plus 3 for the migrate step, 13 per replica), not left to pgx's
// max(4, NumCPU), which moves with the host. An operator overrides either number
// with pool_max_conns in the DSN, so the checks stand down for such a DSN.

func skipIfPoolOverride(t *testing.T, envName string) {
	t.Helper()
	if strings.Contains(os.Getenv(envName), "pool_max_conns") {
		t.Skipf("%s carries its own pool_max_conns", envName)
	}
}

func TestOpen_CapsTheRequestPoolAtTen(t *testing.T) {
	skipIfPoolOverride(t, "DATABASE_URL")
	st := newTestStore(t)
	if got := st.Pool.Config().MaxConns; got != 10 {
		t.Errorf("request pool MaxConns = %d, want 10 (pgx would have taken max(4, NumCPU) = %d here)", got, max(4, runtime.NumCPU()))
	}
}

func TestAttachMigratePool_CapsThePoolAtThree(t *testing.T) {
	ownerURL := os.Getenv("DATABASE_OWNER_URL")
	if ownerURL == "" {
		t.Skip("DATABASE_OWNER_URL not set")
	}
	skipIfPoolOverride(t, "DATABASE_OWNER_URL")
	st := newTestStore(t)
	if err := st.AttachMigratePool(context.Background(), ownerURL); err != nil {
		t.Fatalf("attach owner pool: %v", err)
	}
	if got := st.migratePool.Config().MaxConns; got != 3 {
		t.Errorf("migrate pool MaxConns = %d, want 3 (pgx would have taken max(4, NumCPU) = %d here)", got, max(4, runtime.NumCPU()))
	}
}
