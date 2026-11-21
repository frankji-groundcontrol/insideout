package mcpserver

import (
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// addRoadmapTreeTools registers the task tree (docs/architecture/roadmap-tree.md):
// the tree or one subtree, one node, the caller's own nodes, node edits with
// acceptance (no owner or completion reason), and the owner history. Delegation, hand-back and the
// member list run on routes the server keeps for a human, and every tool acts
// through an agent-scoped token, so the MCP has no tool for them (the CLI and
// the web do).
func addRoadmapTreeTools(ts toolSet) {
	nodeReq := []mcp.ToolOption{strReq("node_id", "roadmap node id")}
	ts.add("roadmap_list", "List a project's roadmap nodes, or with root only that node and the nodes below it. Each node carries canWrite (whether you may write it) and rollup (leaves, done, validated, earliestDeadline)",
		[]mcp.ToolOption{strReq("project_id", "project id"), strOpt("root", "node id: list only this subtree")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.RoadmapList(reqStr(req, "project_id"), optStr(req, "root")))
		})
	ts.add("progress", "Time-first Progress view: Now (deadlined work only), at most three Next, Done count, plus in-progress items missing a deadline; with root, only that subtree",
		[]mcp.ToolOption{strReq("project_id", "project id"), strOpt("root", "node id: Progress of this subtree only")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.RoadmapProgress(reqStr(req, "project_id"), optStr(req, "root")))
		})
	ts.add("roadmap_node", "Read one roadmap node with its ancestors (root first), its project id and name",
		nodeReq,
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.RoadmapNode(reqStr(req, "node_id")))
		})
	ts.add("roadmap_mine", "The signed-in user's own work in a workspace: their topmost owned nodes, each with its project and ancestors",
		[]mcp.ToolOption{strReq("workspace_id", "workspace id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.RoadmapMine(reqStr(req, "workspace_id")))
		})
	// No owner or completion reason: the server refuses both to the agent-scope
	// token every MCP call carries.
	ts.add("roadmap_update", "Partially update a node: title, description, status, deadline, acceptance. An owner change or a completion reason is a human's: file a proposal (propose) for an owner, and move a node to done on evidence",
		[]mcp.ToolOption{strReq("node_id", "node id"), strOpt("title", "new title"),
			strOpt("description", `new description, or "clear"`), strOpt("status", "locked|pending|in_progress|done"),
			strOpt("deadline", `RFC3339 deadline, or "clear"`),
			strOpt("acceptance", `what done means for this node, or "clear"`)},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			// An omitted or empty argument leaves its field alone (apiclient.PatchText).
			patch := apiclient.RoadmapPatch{Title: apiclient.NonEmpty(optStr(req, "title")),
				Description: apiclient.PatchText(optStr(req, "description")), Status: apiclient.NonEmpty(optStr(req, "status")),
				Deadline: apiclient.PatchText(optStr(req, "deadline")), Acceptance: apiclient.PatchText(optStr(req, "acceptance"))}
			return call(c.RoadmapUpdate(reqStr(req, "node_id"), patch))
		})
	ts.add("roadmap_assignments", "Who held a node: delegations, hand-backs and cleared owners, newest first",
		nodeReq,
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.RoadmapAssignments(reqStr(req, "node_id")))
		})
}
