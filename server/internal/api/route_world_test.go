package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

const routeTestSecret = "a-32-byte-or-longer-test-secret!"

// routeUser is a real user row and a session token for it. The zero value is
// a caller with no credentials.
type routeUser struct {
	*store.User
	token string
}

// routeWorld is the cast the DB-gated route tests (TEST-13) share, on real
// PostgreSQL with its row-level security: owner created a workspace and a
// project in it, member joined the workspace, outsider belongs to nothing.
type routeWorld struct {
	st                      *store.Store
	srv                     *Server
	h                       http.Handler
	owner, member, outsider routeUser
	project                 *store.Project
}

// newRouteWorld skips without DATABASE_URL, the same gate as the store suite.
func newRouteWorld(t *testing.T) *routeWorld {
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

	tokens := auth.NewTokenIssuer(routeTestSecret, time.Hour)
	mkUser := func(name string) routeUser {
		u, err := st.CreateUser(ctx, name+"-"+uuid.NewString()+"@test.local", "x", name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		token, err := tokens.MintAccessToken(u.ID)
		if err != nil {
			t.Fatalf("mint %s token: %v", name, err)
		}
		return routeUser{u, token}
	}
	w := &routeWorld{st: st, owner: mkUser("owner"), member: mkUser("member"), outsider: mkUser("outsider")}
	ws, err := st.CreateWorkspace(ctx, w.owner.ID, "Route tests", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, w.member.ID, ws.Code); err != nil {
		t.Fatalf("member joins: %v", err)
	}
	if w.project, err = st.CreateProject(ctx, w.owner.ID, ws.ID, "Route tests", ""); err != nil {
		t.Fatalf("create project: %v", err)
	}
	w.srv = NewServer(st, tokens, &config.Config{PublicURL: "http://app.test"}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	w.h = w.srv.Handler()
	return w
}

// call sends one request as the given user straight into the handler chain
// and returns the status and body.
func (w *routeWorld) call(as routeUser, method, path, body string) (int, string) {
	// The deadline keeps a stream route that lost its guard from hanging the
	// run: it would otherwise hold the recorder open until the test timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	if as.token != "" {
		r.Header.Set("Authorization", "Bearer "+as.token)
	}
	rec := httptest.NewRecorder()
	w.h.ServeHTTP(rec, r)
	return rec.Code, rec.Body.String()
}

// expect fails the test unless a call answered with the wanted status.
func expect(t *testing.T, what string, got int, body string, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: status %d, want %d; body %s", what, got, want, body)
	}
}

// decodeAs parses a response body into T.
func decodeAs[T any](t *testing.T, body string) T {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	return v
}
