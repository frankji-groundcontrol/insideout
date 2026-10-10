package mcpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// workspace_join POSTs the code the human was given.
func TestWorkspaceJoinTool(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.Write([]byte(`{"id":"ws1"}`))
	}))
	t.Cleanup(api.Close)

	res := drive(t, api.URL, []string{
		toolCall(1, "workspace_join", `{"code":"ABC123"}`),
	})
	if res[1].Result.IsError {
		t.Fatalf("workspace_join failed: %+v", res[1])
	}
	if gotMethod != http.MethodPost || gotPath != "/workspaces/join" || gotBody != `{"code":"ABC123"}` {
		t.Errorf("got %s %s body %q, want POST /workspaces/join", gotMethod, gotPath, gotBody)
	}
}

// prd_export returns the export text itself — MCP has no filesystem.
func TestPrdExportTool(t *testing.T) {
	var formats, audiences []string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/prds/p1/export" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		formats = append(formats, r.URL.Query().Get("format"))
		audiences = append(audiences, r.URL.Query().Get("audience"))
		w.Write([]byte("# The PRD"))
	}))
	t.Cleanup(api.Close)

	res := drive(t, api.URL, []string{
		toolCall(1, "prd_export", `{"prd_id":"p1"}`),
		toolCall(2, "prd_export", `{"prd_id":"p1","audience":"delivery"}`),
	})
	if r := res[1]; r.Result.IsError || len(r.Result.Content) == 0 ||
		!strings.Contains(r.Result.Content[0].Text, "# The PRD") {
		t.Errorf("prd_export = %+v, want the markdown text", r)
	}
	if res[2].Result.IsError {
		t.Fatalf("prd_export audience failed: %+v", res[2])
	}
	if len(formats) != 2 || formats[0] != "markdown" || audiences[0] != "" {
		t.Errorf("default call: formats=%v audiences=%v, want markdown with no audience", formats, audiences)
	}
	if len(audiences) != 2 || audiences[1] != "delivery" {
		t.Errorf("audience call: audiences=%v, want delivery", audiences)
	}
}

// roadmap_move must thread an optional position, not only re-parent.
func TestRoadmapMovePassesPosition(t *testing.T) {
	var gotBody string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/roadmap/n1/move" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(api.Close)

	res := drive(t, api.URL, []string{
		toolCall(1, "roadmap_move", `{"node_id":"n1","position":2}`),
	})
	if res[1].Result.IsError {
		t.Fatalf("roadmap_move failed: %+v", res[1])
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(gotBody), &body); err != nil {
		t.Fatalf("body %q: %v", gotBody, err)
	}
	if pos, ok := body["position"].(float64); !ok || pos != 2 {
		t.Errorf("body position = %v, want 2 (body %q)", body["position"], gotBody)
	}
}

// build_confirm must carry expected_count so an agent can confirm a
// replace_conflict 409 retry. build itself writes nothing.
func TestBuildPassesExpectedCount(t *testing.T) {
	var gotBody string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/prds/p1/build/confirm" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.Write([]byte(`{"projectId":"pr1"}`))
	}))
	t.Cleanup(api.Close)

	res := drive(t, api.URL, []string{
		toolCall(1, "build_confirm", `{"prd_id":"p1","tree":"{\"title\":\"Ship\"}","expected_count":5}`),
	})
	if res[1].Result.IsError {
		t.Fatalf("build_confirm failed: %+v", res[1])
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(gotBody), &body); err != nil {
		t.Fatalf("body %q: %v", gotBody, err)
	}
	if n, ok := body["expectedCount"].(float64); !ok || n != 5 {
		t.Errorf("body expectedCount = %v, want 5 (body %q)", body["expectedCount"], gotBody)
	}
}

// toolDescriptions asks the real MCP handler for tools/list and returns each
// tool's description as a client receives it on the wire.
func toolDescriptions(t *testing.T) map[string]string {
	t.Helper()
	msg := Build("http://unused.test").HandleMessage(context.Background(),
		json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("tools/list answer %s: %v", raw, err)
	}
	desc := map[string]string{}
	for _, tool := range out.Result.Tools {
		desc[tool.Name] = tool.Description
	}
	return desc
}

// AIDEV-10: ideas became author-private (2026-10-04) and dropped ideas stopped
// being listed (USER-09), but the descriptions an agent reads from tools/list
// still promised a workspace-wide inbox that keeps dropped ideas around.
func TestIdeaToolDescriptionsMatchAuthorPrivateIdeas(t *testing.T) {
	desc := toolDescriptions(t)
	if d := desc["idea_list"]; strings.Contains(d, "workspace's idea inbox") || !strings.Contains(d, "your own") {
		t.Errorf("idea_list = %q, want it to say it lists the caller's own ideas, not a workspace-wide inbox", d)
	}
	if d := desc["idea_delete"]; d == "" || strings.Contains(d, "stays in idea_list") {
		t.Errorf("idea_delete = %q, but a dropped idea is no longer returned by idea_list", d)
	}
}
