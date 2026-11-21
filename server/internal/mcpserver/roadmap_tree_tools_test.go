package mcpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// rv5-PARC-01, PARC-02, TREEX-04 and the MCP half of TREEB-01, TREEB-03,
// TREEB-04, TREEB-08: the task tree on MCP. roadmap_update carries owner and
// acceptance, and the node, mine, delegation, member and subtree tools each
// send the request the API serves.

// treeRequest is one request a tool made: "METHOD path?query" and its JSON body.
type treeRequest struct {
	target string
	body   map[string]any
}

func treeRecorder(t *testing.T) (*httptest.Server, func() []treeRequest) {
	t.Helper()
	var mu sync.Mutex
	var got []treeRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		target := r.Method + " " + r.URL.Path
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		mu.Lock()
		got = append(got, treeRequest{target, body})
		mu.Unlock()
		if r.Method == http.MethodGet && (r.URL.Path == "/projects/p1" || r.URL.Path == "/workspaces/ws1") {
			w.Write([]byte(`{"id":"x","title":"Old","description":"Old text","status":"active"}`))
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []treeRequest {
		mu.Lock()
		defer mu.Unlock()
		out := got
		got = nil
		return out
	}
}

func TestRoadmapTreeToolsSendTheirRequests(t *testing.T) {
	api, requests := treeRecorder(t)
	for _, tc := range []struct {
		tool, args string
		want       treeRequest
	}{
		{"roadmap_update", `{"node_id":"n1","owner_id":"u2","acceptance":"ships"}`,
			treeRequest{"PATCH /roadmap/n1", map[string]any{"ownerId": "u2", "acceptance": "ships"}}},
		// The CLI's rule (rv5-STRS-06, PARC-07, PARC-09): an absent or empty
		// argument leaves a field alone and the word clear clears it.
		{"roadmap_update", `{"node_id":"n1","owner_id":"clear"}`,
			treeRequest{"PATCH /roadmap/n1", map[string]any{"ownerId": ""}}},
		{"roadmap_update", `{"node_id":"n1","acceptance":"clear","description":"clear"}`,
			treeRequest{"PATCH /roadmap/n1", map[string]any{"acceptance": "", "description": ""}}},
		{"roadmap_update", `{"node_id":"n1","deadline":"clear","completion_reason":"clear"}`,
			treeRequest{"PATCH /roadmap/n1", map[string]any{"deadline": "", "completionReason": ""}}},
		{"roadmap_update", `{"node_id":"n1","title":"T","deadline":"","acceptance":"","owner_id":""}`,
			treeRequest{"PATCH /roadmap/n1", map[string]any{"title": "T"}}},
		{"roadmap_update", `{"node_id":"n1","title":"T"}`,
			treeRequest{"PATCH /roadmap/n1", map[string]any{"title": "T"}}},
		{"roadmap_node", `{"node_id":"n1"}`, treeRequest{"GET /roadmap/n1", nil}},
		{"roadmap_mine", `{"workspace_id":"ws1"}`, treeRequest{"GET /workspaces/ws1/roadmap/mine", nil}},
		{"roadmap_assignments", `{"node_id":"n1"}`, treeRequest{"GET /roadmap/n1/assignments", nil}},
		{"roadmap_list", `{"project_id":"p1","root":"r1"}`, treeRequest{"GET /projects/p1/roadmap?root=r1", nil}},
		{"roadmap_list", `{"project_id":"p1"}`, treeRequest{"GET /projects/p1/roadmap", nil}},
		{"progress", `{"project_id":"p1","root":"r1"}`, treeRequest{"GET /projects/p1/roadmap/progress?root=r1", nil}},
	} {
		res := drive(t, api.URL, []string{toolCall(1, tc.tool, tc.args)})
		got := requests()
		if res[1].Result.IsError {
			t.Errorf("%s %s failed: %+v", tc.tool, tc.args, res[1])
			continue
		}
		if !reflect.DeepEqual(got, []treeRequest{tc.want}) {
			t.Errorf("%s %s sent %v, want %v", tc.tool, tc.args, got, tc.want)
		}
	}
}

// roadmap_update tells an agent that a token of its kind cannot change an
// owner, and what to do instead.
func TestRoadmapUpdateDescribesTheOwnerRefusal(t *testing.T) {
	pinConfigHome(t)
	for _, tool := range listTools(t, Build(exampleAPI)) {
		if tool.Name != "roadmap_update" {
			continue
		}
		owner := tool.InputSchema.Properties["owner_id"]
		if !strings.Contains(owner.Description, "propose") || tool.InputSchema.Properties["acceptance"].Description == "" {
			t.Errorf("roadmap_update owner_id = %q, want the agent-token refusal and propose named; acceptance must be described", owner.Description)
		}
		return
	}
	t.Fatal("roadmap_update is not registered")
}

// INSIDEOUT_AGENT_ROOT confines the agent token the MCP mints to one subtree:
// the exchange asks for that root node.
func TestAgentRootRidesTheTokenExchange(t *testing.T) {
	pinConfigHome(t)
	t.Setenv("INSIDEOUT_TOKEN", humanJWT())
	t.Setenv("INSIDEOUT_AGENT_ROOT", "r1")
	var mint map[string]any
	grant := agentGrant("root", time.Now(), 15*time.Minute)
	api, _ := routedAPI(t, func(w http.ResponseWriter, r *http.Request, _ int) bool {
		if r.URL.Path != "/agent/token" {
			return false
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &mint)
		w.Write([]byte(grant.body()))
		return true
	})
	if r := callTool(t, Build(api.URL), 1, "whoami", `{}`); r.Result.IsError {
		t.Fatalf("whoami: %+v", r)
	}
	if mint["rootNodeId"] != "r1" {
		t.Errorf("POST /agent/token body = %v, want rootNodeId r1", mint)
	}
}
