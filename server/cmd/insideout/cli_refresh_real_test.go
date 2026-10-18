package main

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/api"
	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// Review-4 USER-04 against the real thing: the real /api/v1 handlers over real
// PostgreSQL, with a 2-second access TTL. The saved login has to outlive its
// access token twice in a row, which only works when the refresh token the
// first renewal rotated away reached the file. DATABASE_URL-gated.
func TestSavedLoginOutlivesItsAccessTokenTwice(t *testing.T) {
	const accessTTL = 2 * time.Second
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)

	email, password := "refresh-"+uuid.NewString()+"@example.org", "pw-for-tests-"+uuid.NewString()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := st.CreateUser(ctx, email, hash, "refresh-user"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	cfg := &config.Config{AccessTTL: accessTTL, RefreshTTL: time.Hour}
	srv := api.NewServer(st, auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", accessTTL), cfg,
		slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	c := apiclient.New(ts.URL + "/api/v1")
	if err := c.Login(email, password); err != nil {
		t.Fatalf("login: %v", err)
	}
	writeClientConfig(t, clientconfig.Config{APIBase: ts.URL + "/api/v1", Token: c.Token(), RefreshToken: c.RefreshToken()})

	for round := 1; round <= 2; round++ {
		time.Sleep(accessTTL + time.Second) // the access token is dead now
		res := runCLI(t, cliEnv(), "", "whoami")
		if res.code != 0 || !strings.Contains(res.stdout, email) {
			t.Fatalf("round %d: whoami with an expired access token: exit %d\n%s", round, res.code, res.output())
		}
	}
}
