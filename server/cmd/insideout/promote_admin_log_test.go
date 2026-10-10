package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// REL-10 / REL-23: promote-admin's log can end up in CI or hosted log
// storage, so it must state the fact and not echo the address the operator
// typed. DB-gated: the role change has to land through the real owner pool
// (cross-user updates on users are RLS-blocked for the app role).
func TestRunPromoteAdmin_LogsTheFactNotTheAddress(t *testing.T) {
	url, owner := os.Getenv("DATABASE_URL"), os.Getenv("DATABASE_OWNER_URL")
	if url == "" || owner == "" {
		t.Skip("DATABASE_URL and DATABASE_OWNER_URL not both set, skipping integration test")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	if err := st.AttachMigratePool(ctx, owner); err != nil {
		t.Fatalf("attach owner pool: %v", err)
	}
	email := "promote-" + uuid.NewString() + "@test.local"
	if _, err := st.CreateUser(ctx, email, "x", "promote-tester"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	var logs bytes.Buffer
	runPromoteAdmin(ctx, slog.New(slog.NewTextHandler(&logs, nil)), st, email)

	if !strings.Contains(logs.String(), "promote-admin: done") {
		t.Fatalf("promote-admin did not log completion; log:\n%s", logs.String())
	}
	if strings.Contains(logs.String(), email) {
		t.Fatalf("promote-admin echoed the promoted address into the log:\n%s", logs.String())
	}
	got, err := st.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if got.SystemRole != "admin" {
		t.Fatalf("system_role = %q, want admin", got.SystemRole)
	}
}
