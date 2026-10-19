package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
	"github.com/google/uuid"
)

// rv4-TEST-03: the coverage_*_test.go files drive handlers no test reached
// (project, idea, update, member, PRD and session routes) through the real
// router, real PostgreSQL and its row-level security. Fixtures are made
// through the same routes, so a fixture that breaks is a route that broke, and
// a denied write is judged by what it left behind as well as by its status.

// newCoverageWorld is a routeWorld that, under INSIDEOUT_TEST_STRICT, refuses
// a database that is not on this machine (rv4-TEST-06).
func newCoverageWorld(t *testing.T) *routeWorld {
	t.Helper()
	w := newRouteWorld(t)
	testlock.RequireLocalHost(t, w.st.Pool)
	return w
}

// js is s as a JSON string literal, so a fixture body can carry any text.
func js(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// must sends one request and fails the test unless it answered want. It
// returns the body.
func (w *routeWorld) must(t *testing.T, as routeUser, method, path, body string, want int) string {
	t.Helper()
	status, resp := w.call(as, method, path, body)
	expect(t, method+" "+path, status, resp, want)
	return resp
}

// callRaw is call with the whole response kept, for the tests that read
// headers.
func (w *routeWorld) callRaw(as routeUser, method, path string) *httptest.ResponseRecorder {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r := httptest.NewRequest(method, path, strings.NewReader("")).WithContext(ctx)
	if as.token != "" {
		r.Header.Set("Authorization", "Bearer "+as.token)
	}
	rec := httptest.NewRecorder()
	w.h.ServeHTTP(rec, r)
	return rec
}

// ws and proj are the paths of the world's workspace and project.
func (w *routeWorld) ws() string   { return "/api/v1/workspaces/" + w.project.WorkspaceID.String() }
func (w *routeWorld) proj() string { return "/api/v1/projects/" + w.project.ID.String() }

// newMember is another plain member of the world's workspace: not its admin,
// and the author of nothing the owner made.
func (w *routeWorld) newMember(t *testing.T, name string) routeUser {
	t.Helper()
	ctx := context.Background()
	u, err := w.st.CreateUser(ctx, name+"-"+uuid.NewString()+"@test.local", "x", name)
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	token, err := w.srv.tokens.MintAccessToken(u.ID)
	if err != nil {
		t.Fatalf("mint %s token: %v", name, err)
	}
	ws, err := w.st.GetWorkspaceForMember(ctx, w.project.WorkspaceID, w.owner.ID)
	if err != nil {
		t.Fatalf("read the workspace code: %v", err)
	}
	if _, err := w.st.JoinWorkspace(ctx, u.ID, ws.Code); err != nil {
		t.Fatalf("%s joins: %v", name, err)
	}
	return routeUser{u, token}
}

// fileIdea makes an idea through the route.
func (w *routeWorld) fileIdea(t *testing.T, as routeUser, title string) ideaView {
	t.Helper()
	body := w.must(t, as, "POST", w.ws()+"/ideas", `{"title":`+js(title)+`,"content":"what it is"}`, http.StatusCreated)
	return decodeAs[ideaView](t, body)
}

// note files a progress note on the world's project through the route.
func (w *routeWorld) note(t *testing.T, as routeUser, content string) projectUpdateView {
	t.Helper()
	body := w.must(t, as, "POST", w.proj()+"/updates", `{"kind":"progress","content":`+js(content)+`}`, http.StatusCreated)
	return decodeAs[projectUpdateView](t, body)
}
