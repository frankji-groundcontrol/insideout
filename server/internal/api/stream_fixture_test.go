package api

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// streamFixture is one project with two members (a created it, b joined) and a
// live server: what the stream tests need to act as two different viewers.
// DB-gated, because project membership is enforced by RLS in real PostgreSQL.
type streamFixture struct {
	srv    *Server
	ts     *httptest.Server
	proj   *store.Project
	a, b   *store.User
	tokenA string
	tokenB string
}

func newStreamFixture(t *testing.T) *streamFixture {
	t.Helper()
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
	newUser := func(name string) *store.User {
		u, err := st.CreateUser(ctx, name+"-"+uuid.NewString()+"@test.local", "x", name)
		if err != nil {
			t.Fatalf("create user %s: %v", name, err)
		}
		return u
	}
	a, b := newUser("viewer-a"), newUser("viewer-b")
	ws, err := st.CreateWorkspace(ctx, a.ID, "Streams", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, b.ID, ws.Code); err != nil {
		t.Fatalf("join workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, a.ID, ws.ID, "Streams", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	tokens := auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour)
	mint := func(u *store.User) string {
		tok, err := tokens.MintAccessToken(u.ID)
		if err != nil {
			t.Fatalf("mint token: %v", err)
		}
		return tok
	}
	srv := NewServer(st, tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return &streamFixture{srv: srv, ts: ts, proj: proj, a: a, b: b, tokenA: mint(a), tokenB: mint(b)}
}
