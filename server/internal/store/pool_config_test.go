package store

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
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

// Never a real DSN in these: pgx only parses it, nothing dials db.invalid.
const poolTestDSN = "postgres://app:placeholder@db.invalid:5432/insideout"

func TestPoolConfig_AppliesTheRequestedMaxConns(t *testing.T) {
	// 3 is below pgx's floor of 4, so this cannot pass on its default whatever the host.
	for _, want := range []int32{3, 10} {
		cfg, err := poolConfig(poolTestDSN, want)
		if err != nil {
			t.Fatalf("poolConfig: %v", err)
		}
		if cfg.MaxConns != want {
			t.Errorf("poolConfig(dsn, %d).MaxConns = %d", want, cfg.MaxConns)
		}
	}
}

func TestPoolConfig_APoolMaxConnsInTheDSNWins(t *testing.T) {
	cfg, err := poolConfig(poolTestDSN+"?pool_max_conns=7", 10)
	if err != nil {
		t.Fatalf("poolConfig: %v", err)
	}
	if cfg.MaxConns != 7 {
		t.Errorf("MaxConns = %d, want the DSN's pool_max_conns=7 to win over the default 10", cfg.MaxConns)
	}
}

func TestPoolConfig_PgbouncerModeKeepsTheSimpleProtocol(t *testing.T) {
	cfg, err := poolConfig(poolTestDSN+"?pgbouncer=true", 10)
	if err != nil {
		t.Fatalf("poolConfig: %v", err)
	}
	if cfg.ConnConfig.DefaultQueryExecMode != pgx.QueryExecModeSimpleProtocol {
		t.Errorf("pgbouncer=true: query exec mode = %v, want the simple protocol", cfg.ConnConfig.DefaultQueryExecMode)
	}
	if cfg.MaxConns != 10 {
		t.Errorf("pgbouncer=true: MaxConns = %d, want 10", cfg.MaxConns)
	}
	plain, err := poolConfig(poolTestDSN, 10)
	if err != nil {
		t.Fatalf("poolConfig: %v", err)
	}
	if plain.ConnConfig.DefaultQueryExecMode == pgx.QueryExecModeSimpleProtocol {
		t.Error("a DSN without pgbouncer=true must keep pgx's default protocol")
	}
}
