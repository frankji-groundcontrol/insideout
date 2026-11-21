package mcpserver

import (
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// addRoadmapTreeTools registers the task tree (docs/architecture/roadmap-tree.md):
// the tree or one subtree, one node, the caller's own nodes, node edits with
// owner and acceptance, and the owner history. Delegation, hand-back and the
// member list run on routes the server keeps for a human, and every tool acts
// through an agent-scoped token, so the MCP has no tool for them (the CLI and
// the web do).
func addRoadmapTreeTools(
	addTextTool func(name, desc string, opts []mcp.ToolOption, call func(c *apiclient.Client, req mcp.CallToolRequest) (string, error)),
	strReq, strOpt func(name, desc string) mcp.ToolOption,
) {
	nodeReq := []mcp.ToolOption{strReq("node_id", "roadmap node id")}
	addTextTool("roadmap_list", "List a project's roadmap nodes, or with root only that node and the nodes below it. Each node carries canWrite (whether you may write it) and rollup (leaves, done, validated, earliestDeadline)",
		[]mcp.ToolOption{strReq("project_id", "project id"), strOpt("root", "node id: list only this subtree")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.RoadmapList(reqStr(req, "project_id"), optStr(req, "root")))
		})
	addTextTool("progress", "Time-first Progress view: Now (deadlined work only), at most three Next, Done count, plus in-progress items missing a deadline; with root, only that subtree",
		[]mcp.ToolOption{strReq("project_id", "project id"), strOpt("root", "node id: Progress of this subtree only")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.RoadmapProgress(reqStr(req, "project_id"), optStr(req, "root")))
		})
	addTextTool("roadmap_node", "Read one roadmap node with its ancestors (root first), its project id and name",
		nodeReq,
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.RoadmapNode(reqStr(req, "node_id")))
		})
	addTextTool("roadmap_mine", "The signed-in user's own work in a workspace: their topmost owned nodes, each with its project and ancestors",
		[]mcp.ToolOption{strReq("workspace_id", "workspace id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.RoadmapMine(reqStr(req, "workspace_id")))
		})
	addTextTool("roadmap_update", "Partially update a node: title, description, status, deadline, acceptance, owner, completion reason",
		[]mcp.ToolOption{strReq("node_id", "node id"), strOpt("title", "new title"),
			strOpt("description", "new description"), strOpt("status", "locked|pending|in_progress|done"),
			strOpt("deadline", "RFC3339 deadline, or empty string to clear"),
			strOpt("acceptance", "what done means for this node; an empty string clears it"),
			strOpt("owner_id", `a workspace member's user id, or "none" to clear the owner; omit to keep it. Human tokens only: an agent token is refused an owner change, so file a proposal (propose) for the human instead`),
			strOpt("completion_reason", "one sentence when marking done with no evidence; human tokens only, an agent token is refused (move to done on evidence instead)")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			args := req.GetArguments()
			patch := apiclient.RoadmapPatch{Title: nonEmpty(req, "title"), Description: nonEmpty(req, "description"),
				Status: nonEmpty(req, "status"), CompletionReason: nonEmpty(req, "completion_reason")}
			if v, ok := args["acceptance"].(string); ok {
				patch.Acceptance = &v
			}
			if v, ok := args["deadline"].(string); ok {
				patch.Deadline, patch.ClearDeadline = v, v == ""
			}
			if v := optStr(req, "owner_id"); v == "none" {
				patch.ClearOwner = true
			} else {
				patch.OwnerID = v
			}
			return call(c.RoadmapUpdate(reqStr(req, "node_id"), patch))
		})
	addTextTool("roadmap_assignments", "Who held a node: delegations, hand-backs and cleared owners, newest first",
		nodeReq,
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.RoadmapAssignments(reqStr(req, "node_id")))
		})
}

// optStr reads an optional string argument; absent is "".
func optStr(req mcp.CallToolRequest, name string) string {
	v, _ := req.GetArguments()[name].(string)
	return v
}

// nonEmpty is an optional string argument as a pointer: nil when absent or "".
func nonEmpty(req mcp.CallToolRequest, name string) *string {
	if v := optStr(req, name); v != "" {
		return &v
	}
	return nil
}
