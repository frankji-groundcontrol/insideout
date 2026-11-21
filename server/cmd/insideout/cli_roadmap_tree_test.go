package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
)

// rv5-PARC-01, PARC-02, TREEX-04 and the CLI half of TREEB-01, TREEB-03,
// TREEB-04, TREEB-08: the task tree on the CLI. Owner and acceptance reach
// PATCH, and the subtree, node, mine, delegation and member reads and writes
// each send the request the API serves.

// sent is one request a verb made: "METHOD path?query" and its JSON body.
type sent struct {
	target string
	body   map[string]any
}

// treeAPI records every request and answers GET /me with the user u-me.
func treeAPI(t *testing.T) func() []sent {
	t.Helper()
	var mu sync.Mutex
	var got []sent
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		target := r.Method + " " + r.URL.Path
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		mu.Lock()
		got = append(got, sent{target, body})
		mu.Unlock()
		switch {
		case r.URL.Path == "/me":
			w.Write([]byte(`{"id":"u-me"}`))
		case r.Method == http.MethodGet && (r.URL.Path == "/projects/p1" || r.URL.Path == "/workspaces/ws1"):
			w.Write([]byte(`{"id":"x","title":"Old","description":"Old text","status":"active"}`))
		default:
			w.Write([]byte(`{"ok":true}`))
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("INSIDEOUT_API", srv.URL)
	t.Setenv("INSIDEOUT_TOKEN", "tok")
	return func() []sent {
		mu.Lock()
		defer mu.Unlock()
		out := got
		got = nil
		return out
	}
}

func TestRoadmapTreeVerbsSendTheirRequests(t *testing.T) {
	requests := treeAPI(t)
	for _, tc := range []struct {
		args []string
		want []sent
	}{
		{words("roadmap", "update", "--owner", "u2", "--acceptance", "ships", "n1"),
			[]sent{{"PATCH /roadmap/n1", map[string]any{"ownerId": "u2", "acceptance": "ships"}}}},
		{words("roadmap", "update", "--owner", "clear", "n1"),
			[]sent{{"PATCH /roadmap/n1", map[string]any{"ownerId": ""}}}},
		{words("roadmap", "update", "--owner", "me", "n1"),
			[]sent{{"GET /me", nil}, {"PATCH /roadmap/n1", map[string]any{"ownerId": "u-me"}}}},
		// One rule on the CLI and MCP (rv5-STRS-06, PARC-07, PARC-09): an absent
		// or empty value leaves a field alone and the word clear clears it.
		{words("roadmap", "update", "--acceptance", "clear", "n1"),
			[]sent{{"PATCH /roadmap/n1", map[string]any{"acceptance": ""}}}},
		{words("roadmap", "update", "--description", "clear", "--completion-reason", "clear", "n1"),
			[]sent{{"PATCH /roadmap/n1", map[string]any{"description": "", "completionReason": ""}}}},
		{words("roadmap", "update", "--deadline", "clear", "--acceptance", "", "n1"),
			[]sent{{"PATCH /roadmap/n1", map[string]any{"deadline": ""}}}},
		{words("roadmap", "update", "--title", "T", "n1"),
			[]sent{{"PATCH /roadmap/n1", map[string]any{"title": "T"}}}},
		{words("roadmap", "show", "n1"), []sent{{"GET /roadmap/n1", nil}}},
		{words("roadmap", "mine", "ws1"), []sent{{"GET /workspaces/ws1/roadmap/mine", nil}}},
		{words("roadmap", "delegate", "--to", "u2", "--note", "yours", "n1"),
			[]sent{{"POST /roadmap/n1/delegate", map[string]any{"toUserId": "u2", "note": "yours"}}}},
		{words("roadmap", "delegate", "--to", "u2", "n1"),
			[]sent{{"POST /roadmap/n1/delegate", map[string]any{"toUserId": "u2"}}}},
		{words("roadmap", "handback", "--note", "done here", "n1"),
			[]sent{{"POST /roadmap/n1/handback", map[string]any{"note": "done here"}}}},
		{words("roadmap", "handback", "n1"), []sent{{"POST /roadmap/n1/handback", map[string]any{}}}},
		{words("roadmap", "assignments", "n1"), []sent{{"GET /roadmap/n1/assignments", nil}}},
		{words("roadmap", "list", "--root", "r1", "p1"), []sent{{"GET /projects/p1/roadmap?root=r1", nil}}},
		{words("roadmap", "list", "p1"), []sent{{"GET /projects/p1/roadmap", nil}}},
		{words("roadmap", "progress", "--root", "r1", "p1"), []sent{{"GET /projects/p1/roadmap/progress?root=r1", nil}}},
		{words("members", "ws1"), []sent{{"GET /workspaces/ws1/members", nil}}},
	} {
		var handled bool
		var code int
		captureStdout(t, func() { handled, code = runClientCommand(tc.args) })
		got := requests()
		if !handled || code != 0 {
			t.Errorf("%v = (handled=%v, code=%d), want (true, 0)", tc.args, handled, code)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%v sent %v, want %v", tc.args, got, tc.want)
		}
	}
}

// A delegation needs a target, and a usage error sends nothing.
func TestRoadmapTreeVerbsRefuseMissingArguments(t *testing.T) {
	requests := treeAPI(t)
	for _, args := range [][]string{
		words("roadmap", "delegate", "n1"),
		words("roadmap", "show"),
		words("roadmap", "mine"),
		words("members"),
		words("roadmap", "update", "n1"),
		words("roadmap", "update", "--acceptance", "", "--description", "", "n1"),
	} {
		var handled bool
		var code int
		captureStdout(t, func() { handled, code = runClientCommand(args) })
		if !handled || code != 2 {
			t.Errorf("%v = (handled=%v, code=%d), want a usage error (true, 2)", args, handled, code)
		}
		if got := requests(); len(got) != 0 {
			t.Errorf("%v sent %v on a usage error", args, got)
		}
	}
}
