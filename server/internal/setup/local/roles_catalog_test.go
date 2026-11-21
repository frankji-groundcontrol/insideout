package local

import (
	"context"
	"os"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Same gate as the store integration tests: unset DATABASE_URL skips, and CI's
// skip gate guarantees it is set there. Strict mode refuses another machine
// before the pool dials.
func TestProbeRolesReadsTheCatalog(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	testlock.RequireLocalURL(t, url)
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	checkProbeRoles(t, pool)
}

// rv4-FR-F65: the catalog probe dials DATABASE_URL itself, so strict mode must
// refuse it before it reaches a database on another machine (moved here with
// the test from cmd/insideout's guard list).
func TestRvGuardsStrictModeRefusesARemoteDatabase(t *testing.T) {
	testlock.AssertRefuses(t, "TestProbeRolesReadsTheCatalog")
}
