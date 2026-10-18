package apiclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// rv4-QUAL-03. Ids, codes, formats and audiences reach the request line as a
// person typed them or a model chose them (the MCP tools hand LLM-supplied
// strings straight in). Whatever the value, it must arrive as exactly one path
// segment or one query value: never a second segment, an extra parameter, a
// dropped fragment, or a "." / ".." segment.

const validID = "3f2b8c1e-5a4d-4e6f-9b7a-1c2d3e4f5a6b"

var hostileInputs = []string{
	"../x", "a/b", "a?b=c#d", "a b", "%2e%2e", "..", ".",
	"decision&admin=1#frag", "a+b", "100%", "a\r\nX: y", "日本",
}

// sweepCase is one client method. shape is the request target it must
// produce, with "{h}" wherever the value under test has to land; call puts
// that value into every argument that can reach the URL.
type sweepCase struct {
	name, method, shape string
	call                func(c *Client, h string) error
}

// drop adapts a (result, error) method to the table's error-only call.
func drop(_ any, err error) error { return err }

var emptyJSON = json.RawMessage(`{}`)

// One row per exported Client method (and per distinct URL shape). Query keys
// follow the order each method sends today; valid ids must stay byte-identical.
var sweep = []sweepCase{
	{"Login", "POST", "/auth/login", func(c *Client, h string) error { return c.Login(h, h) }},
	{"Whoami", "GET", "/me", func(c *Client, h string) error { return drop(c.Whoami()) }},
	{"Workspaces", "GET", "/workspaces", func(c *Client, h string) error { return drop(c.Workspaces()) }},
	{"Projects", "GET", "/workspaces/{h}/projects", func(c *Client, h string) error { return drop(c.Projects(h)) }},

	{"Register", "POST", "/auth/register", func(c *Client, h string) error { return c.Register(h, h, h, h) }},
	{"RequestMagicLink", "POST", "/auth/magic-link", func(c *Client, h string) error { return c.RequestMagicLink(h) }},
	{"ConsumeMagicLink", "POST", "/auth/magic-link/consume", func(c *Client, h string) error { return c.ConsumeMagicLink(h) }},
	{"RequestLoginCode", "POST", "/auth/login-code", func(c *Client, h string) error { return c.RequestLoginCode(h) }},
	{"VerifyLoginCode", "POST", "/auth/login-code/verify", func(c *Client, h string) error { return c.VerifyLoginCode(h, h) }},
	{"ForgotPassword", "POST", "/auth/forgot-password", func(c *Client, h string) error { return c.ForgotPassword(h) }},
	{"ResetPassword", "POST", "/auth/reset-password", func(c *Client, h string) error { return c.ResetPassword(h, h) }},
	{"AuthConfig", "GET", "/auth/config", func(c *Client, h string) error { return drop(c.AuthConfig()) }},
	{"CreateInvitation", "POST", "/admin/invitations", func(c *Client, h string) error { return drop(c.CreateInvitation(h, 24)) }},
	{"ListInvitations", "GET", "/admin/invitations", func(c *Client, h string) error { return drop(c.ListInvitations()) }},
	{"RevokeInvitation", "POST", "/admin/invitations/{h}/revoke", func(c *Client, h string) error { return c.RevokeInvitation(h) }},
	{"SetSignupMode", "PUT", "/admin/settings", func(c *Client, h string) error { return drop(c.SetSignupMode(h)) }},

	{"Ideas", "GET", "/workspaces/{h}/ideas", func(c *Client, h string) error { return drop(c.Ideas(h)) }},
	{"UpdateIdea", "PATCH", "/ideas/{h}", func(c *Client, h string) error { return drop(c.UpdateIdea(h, h, h)) }},
	{"DeleteIdea", "DELETE", "/ideas/{h}", func(c *Client, h string) error { return c.DeleteIdea(h) }},
	{"UpdatePrd", "PATCH", "/prds/{h}", func(c *Client, h string) error {
		return drop(c.UpdatePrd(h, &h, map[string]string{"goals": h}))
	}},
	{"SetPrdStatus", "POST", "/prds/{h}/status", func(c *Client, h string) error { return drop(c.SetPrdStatus(h, h)) }},
	{"NodeEvidence", "GET", "/roadmap/{h}/evidence", func(c *Client, h string) error { return drop(c.NodeEvidence(h)) }},
	{"CreateWorkspace", "POST", "/workspaces", func(c *Client, h string) error { return drop(c.CreateWorkspace(h, h)) }},
	{"JoinWorkspace", "POST", "/workspaces/join", func(c *Client, h string) error { return drop(c.JoinWorkspace(h)) }},
	{"CreateProject", "POST", "/workspaces/{h}/projects", func(c *Client, h string) error { return drop(c.CreateProject(h, h, h)) }},

	{"IdeaCreate", "POST", "/workspaces/{h}/ideas", func(c *Client, h string) error { return drop(c.IdeaCreate(h, h, h)) }},
	{"IdeaConvert", "POST", "/ideas/{h}/convert", func(c *Client, h string) error { return drop(c.IdeaConvert(h)) }},
	{"Prd", "GET", "/prds/{h}", func(c *Client, h string) error { return drop(c.Prd(h)) }},
	{"PrdRevisions", "GET", "/prds/{h}/revisions", func(c *Client, h string) error { return drop(c.PrdRevisions(h)) }},
	{"SnapshotPrd", "POST", "/prds/{h}/revisions", func(c *Client, h string) error { return drop(c.SnapshotPrd(h, h)) }},
	{"CommitPrd", "POST", "/prds/{h}/commit", func(c *Client, h string) error {
		return drop(c.CommitPrd(h, h, h, h, []string{h}, h))
	}},
	{"PrdVersions", "GET", "/prds/{h}/commits", func(c *Client, h string) error { return drop(c.PrdVersions(h)) }},
	{"PrdReadiness", "GET", "/prds/{h}/readiness", func(c *Client, h string) error { return drop(c.PrdReadiness(h)) }},
	{"PrdView", "GET", "/prds/{h}/view?audience={h}", func(c *Client, h string) error { return drop(c.PrdView(h, h)) }},
	{"PrdExport", "GET", "/prds/{h}/export?audience={h}&format=markdown", func(c *Client, h string) error {
		return drop(c.PrdExport(h, "", h))
	}},
	{"PrdExport", "GET", "/prds/{h}/export?audience={h}&format={h}", func(c *Client, h string) error {
		return drop(c.PrdExport(h, h, h))
	}},
	{"PrdExportAudience", "GET", "/prds/{h}/export?format=markdown&audience={h}", func(c *Client, h string) error {
		return drop(c.PrdExportAudience(h, h))
	}},
	{"AgentContext", "GET", "/agent/context?project_id={h}&mode={h}&focus={h}", func(c *Client, h string) error {
		return drop(c.AgentContext(h, h, h))
	}},
	{"AgentContext", "GET", "/agent/context?project_id={h}", func(c *Client, h string) error {
		return drop(c.AgentContext(h, "", ""))
	}},
	{"AgentCheckpoint", "POST", "/agent/checkpoint", func(c *Client, h string) error { return drop(c.AgentCheckpoint(h, h, h)) }},
	{"AgentPropose", "POST", "/agent/propose", func(c *Client, h string) error {
		return drop(c.AgentPropose(h, h, h, h, []ProposeItem{{Title: h, ParentHint: h}}))
	}},
	{"DecideProposal", "POST", "/agent/proposals/{h}/decision", func(c *Client, h string) error {
		return drop(c.DecideProposal(h, "accept", h, true))
	}},

	{"BuildFromPrd", "POST", "/prds/{h}/build", func(c *Client, h string) error { return drop(c.BuildFromPrd(h)) }},
	{"ConfirmBuildFromPrd", "POST", "/prds/{h}/build/confirm", func(c *Client, h string) error {
		return drop(c.ConfirmBuildFromPrd(h, emptyJSON, 3))
	}},
	{"ExpandNode", "POST", "/roadmap/{h}/expand", func(c *Client, h string) error { return drop(c.ExpandNode(h)) }},
	{"ConfirmExpandNode", "POST", "/roadmap/{h}/expand/confirm", func(c *Client, h string) error {
		return drop(c.ConfirmExpandNode(h, emptyJSON))
	}},
	{"RoadmapProgress", "GET", "/projects/{h}/roadmap/progress", func(c *Client, h string) error { return drop(c.RoadmapProgress(h)) }},
	{"RoadmapList", "GET", "/projects/{h}/roadmap", func(c *Client, h string) error { return drop(c.RoadmapList(h)) }},
	{"RoadmapAdd", "POST", "/projects/{h}/roadmap", func(c *Client, h string) error { return drop(c.RoadmapAdd(h, h, h, &h)) }},
	{"RoadmapUpdate", "PATCH", "/roadmap/{h}", func(c *Client, h string) error {
		return drop(c.RoadmapUpdate(h, &h, &h, &h, &h))
	}},
	{"RoadmapMove", "POST", "/roadmap/{h}/move", func(c *Client, h string) error { return drop(c.RoadmapMove(h, &h, nil)) }},
	{"RoadmapDelete", "DELETE", "/roadmap/{h}", func(c *Client, h string) error { return c.RoadmapDelete(h) }},
	{"SetRepo", "PUT", "/projects/{h}/repo", func(c *Client, h string) error { return drop(c.SetRepo(h, h)) }},
	{"SyncGithub", "POST", "/projects/{h}/sync-github", func(c *Client, h string) error { return drop(c.SyncGithub(h)) }},
	{"Guide", "GET", "/projects/{h}/guide", func(c *Client, h string) error { return drop(c.Guide(h)) }},
	{"Cursor", "POST", "/projects/{h}/cursor", func(c *Client, h string) error {
		// The real route answers 204; this server's JSON body only trips
		// Cursor's nil decoder after the request was recorded, and what is
		// under test is the request, not the reply.
		_ = c.Cursor(h, 1, 2)
		return nil
	}},
	{"ProjectPresence", "GET", "/projects/{h}/presence", func(c *Client, h string) error { return drop(c.ProjectPresence(h)) }},
}

// recorder starts a server that answers every request with a token-bearing
// JSON body and returns the client plus a drain of the "METHOD target" lines
// received since the previous drain. RequestURI is the wire form: r.URL.Path
// is already decoded and would hide the difference.
func recorder(t *testing.T) (*Client, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Method+" "+r.RequestURI)
		mu.Unlock()
		w.Write([]byte(`{"accessToken":"tok"}`))
	})
	c.SetToken("tok")
	return c, func() []string {
		mu.Lock()
		defer mu.Unlock()
		out := seen
		seen = nil
		return out
	}
}

// checkTarget asserts that target is shape with every "{h}" replaced by one
// path segment or one query value that decodes back to exactly h, and nothing
// else added.
func checkTarget(t *testing.T, shape, target, h string) {
	t.Helper()
	wantPath, wantQuery, _ := strings.Cut(shape, "?")
	gotPath, gotQuery, _ := strings.Cut(target, "?")

	wantSegs, gotSegs := strings.Split(wantPath, "/"), strings.Split(gotPath, "/")
	if len(gotSegs) != len(wantSegs) {
		t.Errorf("h=%q: path %q has %d segments, want %d (%s)", h, gotPath, len(gotSegs), len(wantSegs), wantPath)
		return
	}
	for i, rawSeg := range gotSegs {
		want := wantSegs[i]
		if want == "{h}" {
			want = h
		}
		got, err := url.PathUnescape(rawSeg)
		if err != nil || got != want {
			t.Errorf("h=%q: path segment %d = %q (decodes to %q), want exactly %q", h, i, rawSeg, got, want)
		}
		if rawSeg == "." || rawSeg == ".." {
			t.Errorf("h=%q: path %q carries the dot-segment %q", h, gotPath, rawSeg)
		}
	}

	gotQ, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Errorf("h=%q: query %q does not parse: %v", h, gotQuery, err)
		return
	}
	wantQ := map[string]string{}
	for _, pair := range strings.Split(wantQuery, "&") {
		if pair == "" {
			continue
		}
		k, v, _ := strings.Cut(pair, "=")
		if v == "{h}" {
			v = h
		}
		wantQ[k] = v
	}
	if len(gotQ) != len(wantQ) {
		t.Errorf("h=%q: query %q has keys %v, want %d keys %v", h, gotQuery, gotQ, len(wantQ), wantQ)
		return
	}
	for k, v := range wantQ {
		if g := gotQ[k]; len(g) != 1 || g[0] != v {
			t.Errorf("h=%q: query %q: %s = %q, want exactly %q", h, gotQuery, k, g, v)
		}
	}
}

// Every method, every hostile value: the value stays one segment or one query
// value. A valid id must also keep the exact URL it always had.
func TestHostileValuesStayOneSegmentOrOneQueryValue(t *testing.T) {
	c, drain := recorder(t)
	for _, tc := range sweep {
		t.Run(tc.name, func(t *testing.T) {
			for _, h := range append([]string{validID}, hostileInputs...) {
				drain()
				if err := tc.call(c, h); err != nil {
					t.Errorf("h=%q: %v", h, err)
					continue
				}
				got := drain()
				if len(got) != 1 {
					t.Errorf("h=%q: server saw %d requests, want 1: %q", h, len(got), got)
					continue
				}
				method, target, _ := strings.Cut(got[0], " ")
				if method != tc.method {
					t.Errorf("h=%q: method %s, want %s", h, method, tc.method)
				}
				checkTarget(t, tc.shape, target, h)
				if want := strings.ReplaceAll(tc.shape, "{h}", validID); h == validID && target != want {
					t.Errorf("valid id: target %q, want byte-identical %q", target, want)
				}
			}
		})
	}
}

// The sweep is only "every method" while it lists every method: a new Client
// method without a row, or a row naming a method that is gone, fails here.
func TestSweepCoversEveryClientMethod(t *testing.T) {
	typ := reflect.TypeOf(&Client{})
	covered := map[string]bool{"SetToken": true, "Token": true} // accessors, no request
	for _, tc := range sweep {
		covered[tc.name] = true
		if _, ok := typ.MethodByName(tc.name); !ok {
			t.Errorf("sweep row %q names no Client method", tc.name)
		}
	}
	for i := 0; i < typ.NumMethod(); i++ {
		if name := typ.Method(i).Name; !covered[name] {
			t.Errorf("Client.%s has no row in the escape sweep: add one", name)
		}
	}
}

// What the sweep proves on the wire, this proves through a real router (the
// Go 1.22 ServeMux the API itself uses): ".." must not climb onto another
// route under the caller's bearer, and an audience must not smuggle a second
// query parameter.
func TestHostileValuesReachTheirOwnRoute(t *testing.T) {
	var mu sync.Mutex
	var last [3]string // route, path wildcard, raw query
	note := func(route string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			last = [3]string{route, r.PathValue("id"), r.URL.RawQuery}
			mu.Unlock()
			w.Write([]byte(`{}`))
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /me", note("GET /me"))
	mux.HandleFunc("GET /prds/{id}", note("GET /prds/{id}"))
	mux.HandleFunc("GET /prds/{id}/view", note("GET /prds/{id}/view"))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c := New(srv.URL)
	c.SetToken("tok")
	got := func() [3]string {
		mu.Lock()
		defer mu.Unlock()
		return last
	}

	if _, err := c.Prd("../me"); err != nil {
		t.Fatalf("Prd: %v", err)
	}
	if g := got(); g[0] != "GET /prds/{id}" || g[1] != "../me" {
		t.Errorf("Prd(%q) was answered by %q with id %q, want GET /prds/{id} with the whole value as id", "../me", g[0], g[1])
	}

	audience := "decision&admin=1#frag"
	if _, err := c.PrdView("p1", audience); err != nil {
		t.Fatalf("PrdView: %v", err)
	}
	g := got()
	q, err := url.ParseQuery(g[2])
	if err != nil {
		t.Fatalf("query %q: %v", g[2], err)
	}
	if g[0] != "GET /prds/{id}/view" || g[1] != "p1" || len(q) != 1 || q.Get("audience") != audience {
		t.Errorf("PrdView reached %q id %q query %v, want one audience=%q", g[0], g[1], q, audience)
	}
}
