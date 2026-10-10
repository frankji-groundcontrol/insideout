package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/agent"
	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// SEC-07: avatarUrl (PATCH /me) and coverUrl (PATCH /workspaces/{id}) are
// user-supplied display URLs echoed to other members. They must validate
// their scheme at the door so a stored javascript:/data: URL can never
// reach a future surface that renders them as links.
type urlSchemeFixture struct {
	st     *store.Store
	ts     *httptest.Server
	ws     *store.Workspace
	owner  *store.User
	tokens *auth.TokenIssuer
}

func newURLSchemeFixture(t *testing.T) urlSchemeFixture {
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

	owner, err := st.CreateUser(ctx, "scheme-"+uuid.NewString()+"@test.local", "x", "scheme")
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, owner.ID, "Scheme door", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	tokens := auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour)
	srv := NewServer(st, tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, agent.NewTemplateRoadmapPlanner())
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return urlSchemeFixture{st: st, ts: ts, ws: ws, owner: owner, tokens: tokens}
}

func (f urlSchemeFixture) patchMe(t *testing.T, body string) int {
	t.Helper()
	token, err := f.tokens.MintAccessToken(f.owner.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	resp := doJSON(t, f.ts, token, http.MethodPatch, "/api/v1/me", body)
	defer resp.Body.Close()
	return resp.StatusCode
}

func (f urlSchemeFixture) patchWorkspace(t *testing.T, body string) int {
	t.Helper()
	token, err := f.tokens.MintAccessToken(f.owner.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	resp := doJSON(t, f.ts, token, http.MethodPatch, "/api/v1/workspaces/"+f.ws.ID.String(), body)
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestUpdateMeAvatarURLSchemeDoor(t *testing.T) {
	f := newURLSchemeFixture(t)

	for _, bad := range []string{"javascript:alert(1)", "data:text/html;base64,PHNjcmlwdD4=", "ftp://example.com/a.png"} {
		body := `{"username":"scheme","avatarUrl":"` + bad + `"}`
		if status := f.patchMe(t, body); status != http.StatusBadRequest {
			t.Fatalf("PATCH /me avatarUrl %q: status %d, want 400", bad, status)
		}
	}
	if status := f.patchMe(t, `{"username":"scheme","avatarUrl":"https://cdn.example.com/a.png"}`); status != http.StatusOK {
		t.Fatalf("PATCH /me https avatarUrl: status %d, want 200", status)
	}
}

func TestUpdateWorkspaceCoverURLSchemeDoor(t *testing.T) {
	f := newURLSchemeFixture(t)

	for _, bad := range []string{"javascript:alert(1)", "data:text/html;base64,PHNjcmlwdD4=", "ftp://example.com/a.png"} {
		body := `{"title":"Scheme door","coverUrl":"` + bad + `"}`
		if status := f.patchWorkspace(t, body); status != http.StatusBadRequest {
			t.Fatalf("PATCH workspace coverUrl %q: status %d, want 400", bad, status)
		}
	}
	if status := f.patchWorkspace(t, `{"title":"Scheme door","coverUrl":"https://cdn.example.com/c.png"}`); status != http.StatusOK {
		t.Fatalf("PATCH workspace https coverUrl: status %d, want 200", status)
	}
}
