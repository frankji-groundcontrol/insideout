package main

import (
	"reflect"
	"testing"
)

// rv5-PARA-06: the workspace, member, update and profile writes the web had
// alone, on the CLI. A delete of a project or a workspace takes everything
// under it, so it asks for --yes. A profile edit sends what it was not given
// as it was read (the server rewrites username, bio and keywords on every
// PATCH /me), leaves the avatar out unless told, and the word clear clears.
func TestPeopleVerbsSendTheirRequests(t *testing.T) {
	requests := treeAPI(t)
	me := func(kv ...any) map[string]any {
		m := map[string]any{"username": "Me", "bio": "Old bio", "keywords": []any{"go"}}
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
		return m
	}
	noKeywords := me() // an empty list is sent as absent, which the server stores as none
	delete(noKeywords, "keywords")
	for _, tc := range []struct {
		args []string
		want []sent
	}{
		{words("project", "delete", "--yes", "p1"), []sent{{"DELETE /projects/p1", nil}}},
		{words("workspace", "delete", "--yes", "ws1"), []sent{{"DELETE /workspaces/ws1", nil}}},
		{words("member", "role", "--role", "admin", "ws1", "u2"),
			[]sent{{"PATCH /workspaces/ws1/members/u2", map[string]any{"role": "admin"}}}},
		{words("member", "remove", "ws1", "u2"), []sent{{"DELETE /workspaces/ws1/members/u2", nil}}},
		{words("update", "edit", "--content", "fixed", "u1"), []sent{{"PATCH /updates/u1", map[string]any{"content": "fixed"}}}},
		{words("update", "delete", "u1"), []sent{{"DELETE /updates/u1", nil}}},
		{words("profile", "update", "--bio", "New bio"), []sent{{"GET /me", nil}, {"PATCH /me", me("bio", "New bio")}}},
		{words("profile", "update", "--username", "Neo", "--bio", "clear", "--avatar", "clear"),
			[]sent{{"GET /me", nil}, {"PATCH /me", me("username", "Neo", "bio", "", "avatarUrl", "")}}},
		{words("profile", "update", "--avatar", "https://img.example/b.png"),
			[]sent{{"GET /me", nil}, {"PATCH /me", me("avatarUrl", "https://img.example/b.png")}}},
		// rv5-PARC-03: the parity walk found keywords reachable from no CLI verb.
		{words("profile", "update", "--keywords", "go, web"),
			[]sent{{"GET /me", nil}, {"PATCH /me", me("keywords", []any{"go", "web"})}}},
		{words("profile", "update", "--keywords", "clear"), []sent{{"GET /me", nil}, {"PATCH /me", noKeywords}}},
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

// A usage error sends nothing: a delete without --yes, a role change with no
// role, a member verb with one id, an edit with nothing to change.
func TestPeopleVerbsRefuseMissingArguments(t *testing.T) {
	requests := treeAPI(t)
	for _, args := range [][]string{
		words("project", "delete", "p1"),
		words("workspace", "delete", "ws1"),
		words("member", "role", "ws1", "u2"),
		words("member", "remove", "ws1"),
		words("member"),
		words("update", "edit", "u1"),
		words("update", "delete"),
		words("profile", "update"),
		words("profile", "update", "--bio", ""),
		words("profile"),
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
