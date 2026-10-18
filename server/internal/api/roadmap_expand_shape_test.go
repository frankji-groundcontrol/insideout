package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/agent"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
)

// rv4-FULL-01: the AI expand route is a two-step contract. POST
// /roadmap/{id}/expand drafts subtasks and writes nothing (200
// {"children":[...]}); POST /roadmap/{id}/expand/confirm writes the drafted
// subtasks (201, a JSON list of roadmap nodes). The Flutter client once read
// the draft as the node list and applied it itself, so this test pins what the
// server really sends against the two captured responses in testdata/, which
// client/test/roadmap_expand_contract_test.dart parses with the real client.
// Real PostgreSQL through the whole handler chain, the planner is the built-in
// template one (no network).

// shapeOf reduces a decoded JSON value to its structure: object keys with the
// kind of every leaf, arrays by the distinct shapes of their elements. Values
// never matter, so a response captured once still describes every later run.
func shapeOf(v any) string {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = k + ":" + shapeOf(x[k])
		}
		return "{" + strings.Join(parts, ",") + "}"
	case []any:
		seen := map[string]bool{}
		var shapes []string
		for _, e := range x {
			if s := shapeOf(e); !seen[s] {
				seen[s] = true
				shapes = append(shapes, s)
			}
		}
		sort.Strings(shapes)
		return "[" + strings.Join(shapes, "|") + "]"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "bool"
	case nil:
		return "null"
	}
	return fmt.Sprintf("%T", v)
}

// fixtureJSON reads a captured response from testdata/ as decoded JSON.
func fixtureJSON(t *testing.T, name string) (raw []byte, decoded any) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("fixture %s is not JSON: %v", name, err)
	}
	return raw, decoded
}

func TestExpandDraftAndConfirmKeepTheirCapturedShapes(t *testing.T) {
	w := newRouteWorld(t)
	// The route world has no planner (expand answers 503); this contract needs
	// the real route chain with the built-in template planner behind it.
	w.srv = NewServer(w.st, w.srv.tokens, &config.Config{PublicURL: "http://app.test"},
		slog.New(slog.NewTextHandler(io.Discard, nil)), nil, agent.NewTemplateRoadmapPlanner())
	w.h = w.srv.Handler()
	nodeID := w.roadmapNode(t, "Ship the first version")
	expand := "/api/v1/roadmap/" + nodeID + "/expand"

	// Step one: a draft. Shaped like the captured response, and it writes nothing.
	status, body := w.call(w.owner, http.MethodPost, expand, "")
	expect(t, "expand draft", status, body, http.StatusOK)
	_, wantDraft := fixtureJSON(t, "roadmap_expand_draft.json")
	if got, want := shapeOf(decodeAs[any](t, body)), shapeOf(wantDraft); got != want {
		t.Fatalf("the expand draft changed shape\n got  %s\n want %s\n (testdata/roadmap_expand_draft.json is what the client is tested against)", got, want)
	}
	if n := len(w.roadmapTitles(t)); n != 1 {
		t.Fatalf("drafting subtasks wrote nodes: the project has %d, want only the parent", n)
	}

	// Step two: confirm the draft as received. 201 and a list of nodes.
	drafted := decodeAs[struct{ Children []json.RawMessage }](t, body).Children
	confirmBody, err := json.Marshal(map[string]any{"children": drafted})
	if err != nil {
		t.Fatalf("encode confirm body: %v", err)
	}
	status, body = w.call(w.owner, http.MethodPost, expand+"/confirm", string(confirmBody))
	expect(t, "expand confirm", status, body, http.StatusCreated)
	_, wantConfirm := fixtureJSON(t, "roadmap_expand_confirm.json")
	if got, want := shapeOf(decodeAs[any](t, body)), shapeOf(wantConfirm); got != want {
		t.Fatalf("the expand confirm response changed shape\n got  %s\n want %s\n (testdata/roadmap_expand_confirm.json is what the client is tested against)", got, want)
	}
	if created := decodeAs[[]roadmapNodeView](t, body); len(created) != len(drafted) {
		t.Fatalf("confirm created %d nodes for %d drafted subtasks", len(created), len(drafted))
	}
	if n := len(w.roadmapTitles(t)); n != 1+len(drafted) {
		t.Fatalf("after confirm the project has %d nodes, want %d", n, 1+len(drafted))
	}
}

// roadmapTitles lists the titles of the world's project roadmap.
func (w *routeWorld) roadmapTitles(t *testing.T) []string {
	t.Helper()
	status, body := w.call(w.owner, http.MethodGet, "/api/v1/projects/"+w.project.ID.String()+"/roadmap", "")
	expect(t, "list roadmap", status, body, http.StatusOK)
	var titles []string
	for _, n := range decodeAs[[]roadmapNodeView](t, body) {
		titles = append(titles, n.Title)
	}
	return titles
}
