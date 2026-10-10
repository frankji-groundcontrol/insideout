package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// USER-06: workspace titles are not unique (only the join code is), so a
// second `seed` used to add another demo workspace, project, idea and PRD.
// DB-gated and run against a throwaway user so it never touches the real
// demo login.
func TestSeedDemo_RerunDoesNotDuplicateTheDemoWorkspace(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	email := "seed-" + uuid.NewString() + "@test.local"
	first, err := seedDemo(ctx, log, st, email)
	if err != nil {
		t.Fatalf("first seed: %v", err)
	}
	second, err := seedDemo(ctx, log, st, email)
	if err != nil {
		t.Fatalf("second seed: %v", err)
	}

	user, err := st.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get demo user: %v", err)
	}
	workspaces, err := st.ListWorkspacesForUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("list workspaces: %v", err)
	}
	if len(workspaces) != 1 || workspaces[0].Title != "InsideOut Demo Team" {
		t.Fatalf("after two seeds the demo user has %d workspaces, want exactly one %q", len(workspaces), "InsideOut Demo Team")
	}
	if !first || second {
		t.Fatalf("seeded = (%v, %v), want (true, false): the rerun must report it skipped", first, second)
	}
}
