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
