package mcpserver

import (
	"reflect"
	"testing"
)

// rv5-PARB-07, PARB-03: the paged updates feed and a workspace's PRD list on
// MCP, both on routes an agent token may read.
func TestProjectReadToolsSendTheirRequests(t *testing.T) {
	api, requests := treeRecorder(t)
	for _, tc := range []struct {
		tool, args string
		want       treeRequest
	}{
		{"project_updates", `{"project_id":"p1"}`, treeRequest{"GET /projects/p1/updates", nil}},
		{"project_updates", `{"project_id":"p1","before":"c1","limit":2}`, treeRequest{"GET /projects/p1/updates?before=c1&limit=2", nil}},
		{"project_updates", `{"project_id":"p1","node_id":"n1","pending":true}`, treeRequest{"GET /projects/p1/updates?node=n1&pending=true", nil}},
		{"prd_list", `{"workspace_id":"ws1"}`, treeRequest{"GET /workspaces/ws1/prds", nil}},
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

// rv5-PARB-04, PARA-06: one project and one workspace, read and edited, and a
// project's updates posted and corrected, on MCP. An edit sends what it was not
// given as it was read (the server rewrites title, description and status on
// every PATCH); an empty value leaves a field, the word clear clears it. There
// is no owner argument: an agent token may not change a project's owner.
func TestProjectWriteToolsSendTheirRequests(t *testing.T) {
	api, requests := treeRecorder(t)
	with := func(k string, v any) map[string]any {
		m := map[string]any{"title": "Old", "description": "Old text", "status": "active"}
		m[k] = v
		return m
	}
	for _, tc := range []struct {
		tool, args string
		want       []treeRequest
	}{
		{"project", `{"project_id":"p1"}`, []treeRequest{{"GET /projects/p1", nil}}},
		{"workspace", `{"workspace_id":"ws1"}`, []treeRequest{{"GET /workspaces/ws1", nil}}},
		{"project_update", `{"project_id":"p1","status":"paused","description":""}`,
			[]treeRequest{{"GET /projects/p1", nil}, {"PATCH /projects/p1", with("status", "paused")}}},
		{"project_update", `{"project_id":"p1","description":"clear"}`,
			[]treeRequest{{"GET /projects/p1", nil}, {"PATCH /projects/p1", with("description", "")}}},
		{"workspace_update", `{"workspace_id":"ws1","cover_url":"clear"}`,
			[]treeRequest{{"GET /workspaces/ws1", nil}, {"PATCH /workspaces/ws1", map[string]any{"title": "Old", "description": "Old text", "coverUrl": ""}}}},
		{"update_add", `{"project_id":"p1","kind":"blocker","content":"stuck"}`,
			[]treeRequest{{"POST /projects/p1/updates", map[string]any{"kind": "blocker", "content": "stuck"}}}},
		{"update_edit", `{"update_id":"u1","content":"unstuck"}`,
			[]treeRequest{{"PATCH /updates/u1", map[string]any{"content": "unstuck"}}}},
	} {
		res := drive(t, api.URL, []string{toolCall(1, tc.tool, tc.args)})
		got := requests()
		if res[1].Result.IsError {
			t.Errorf("%s %s failed: %+v", tc.tool, tc.args, res[1])
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s %s sent %v, want %v", tc.tool, tc.args, got, tc.want)
		}
	}

	// Nothing to change is an error that sends nothing.
	res := drive(t, api.URL, []string{toolCall(1, "project_update", `{"project_id":"p1","title":""}`)})
	if got := requests(); !res[1].Result.IsError || len(got) != 0 {
		t.Errorf("project_update with nothing to change: error=%v sent %v, want an error and no request", res[1].Result.IsError, got)
	}
}
