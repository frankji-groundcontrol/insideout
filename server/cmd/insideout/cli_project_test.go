package main

import (
	"reflect"
	"testing"
)

// rv5-PARB-04, PARA-06, PARB-07, PARB-03: a project, its updates, a workspace
// and a workspace's PRDs on the CLI. An edit sends what it was not given as it
// was read (the server rewrites title, description and status on every PATCH),
// and the word clear clears.
func TestProjectVerbsSendTheirRequests(t *testing.T) {
	requests := treeAPI(t)
	old := map[string]any{"title": "Old", "description": "Old text", "status": "active"}
	with := func(kv ...any) map[string]any {
		m := map[string]any{}
		for k, v := range old {
			m[k] = v
		}
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
		return m
	}
	for _, tc := range []struct {
		args []string
		want []sent
	}{
		{words("project", "p1"), []sent{{"GET /projects/p1", nil}}},
		{words("project", "update", "--status", "paused", "p1"),
			[]sent{{"GET /projects/p1", nil}, {"PATCH /projects/p1", with("status", "paused")}}},
		{words("project", "update", "--title", "New", "--description", "clear", "--owner", "clear", "p1"),
			[]sent{{"GET /projects/p1", nil}, {"PATCH /projects/p1", with("title", "New", "description", "", "ownerId", "")}}},
		{words("project", "update", "--owner", "me", "p1"),
			[]sent{{"GET /me", nil}, {"GET /projects/p1", nil}, {"PATCH /projects/p1", with("ownerId", "u-me")}}},
		{words("updates", "p1"), []sent{{"GET /projects/p1/updates", nil}}},
		{words("updates", "--before", "c1", "--limit", "2", "p1"), []sent{{"GET /projects/p1/updates?before=c1&limit=2", nil}}},
		{words("updates", "--node", "n1", "--pending", "p1"), []sent{{"GET /projects/p1/updates?node=n1&pending=true", nil}}},
		{words("update", "add", "--kind", "blocker", "--content", "stuck", "p1"),
			[]sent{{"POST /projects/p1/updates", map[string]any{"kind": "blocker", "content": "stuck"}}}},
		{words("workspace", "ws1"), []sent{{"GET /workspaces/ws1", nil}}},
		{words("workspace", "update", "--description", "clear", "--cover", "https://img.example/c.png", "ws1"),
			[]sent{{"GET /workspaces/ws1", nil}, {"PATCH /workspaces/ws1", map[string]any{"title": "Old", "description": "", "coverUrl": "https://img.example/c.png"}}}},
		{words("workspace", "update", "--cover", "clear", "ws1"),
			[]sent{{"GET /workspaces/ws1", nil}, {"PATCH /workspaces/ws1", map[string]any{"title": "Old", "description": "Old text", "coverUrl": ""}}}},
		{words("prds", "ws1"), []sent{{"GET /workspaces/ws1/prds", nil}}},
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

// A usage error sends nothing: an edit with nothing to change, an update with
// no kind or content, a verb with no id.
func TestProjectVerbsRefuseMissingArguments(t *testing.T) {
	requests := treeAPI(t)
	for _, args := range [][]string{
		words("project"),
		words("project", "update", "p1"),
		words("project", "delete"),
		words("updates"),
		words("update", "add", "--content", "x", "p1"),
		words("update", "add", "--kind", "note", "p1"),
		words("update"),
		words("workspace", "update", "ws1"),
		words("prds"),
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
