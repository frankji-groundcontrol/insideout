package mcpserver

import (
	"errors"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// errNothingToChange answers an edit tool given no field to change.
var errNothingToChange = errors.New("nothing to change: pass at least one field (the word clear clears one)")

// addProjectTools registers the project list, create, read and edit, and who is
// viewing one (rv5-PARB-04, PARA-06). The edit fills the CLI's request struct
// by its rule: an absent or empty value leaves a field, the word clear clears
// it. project_update takes no owner (an agent token may not change an owner),
// and deleting a project stays a human's.
func addProjectTools(ts toolSet) {
	ts.add("projects", "List projects in a workspace", []mcp.ToolOption{strReq("workspace_id", "workspace id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.Projects(reqStr(req, "workspace_id")))
		})
	ts.add("project_create", "Create a project inside a workspace",
		[]mcp.ToolOption{strReq("workspace_id", "workspace id"), strReq("title", "project title"), strOpt("description", "project description")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.CreateProject(reqStr(req, "workspace_id"), reqStr(req, "title"), optStr(req, "description")))
		})
	ts.add("project", "Get one project with its newest page of updates and their nextCursor",
		[]mcp.ToolOption{strReq("project_id", "project id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.Project(reqStr(req, "project_id")))
		})
	ts.add("project_update", "Edit a project's title, description or status; a field left out keeps its value, the word clear clears the description",
		[]mcp.ToolOption{strReq("project_id", "project id"), strOpt("title", "new title"),
			strOpt("description", "new description, or clear"), strOpt("status", "planning|active|paused|done|archived")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			patch := apiclient.ProjectPatch{Title: apiclient.NonEmpty(optStr(req, "title")),
				Description: apiclient.PatchText(optStr(req, "description")), Status: apiclient.NonEmpty(optStr(req, "status"))}
			if patch == (apiclient.ProjectPatch{}) {
				return "", errNothingToChange
			}
			return call(patch.Apply(c, reqStr(req, "project_id")))
		})
	ts.add("presence", "Who is currently viewing a project (canvas presence snapshot)",
		[]mcp.ToolOption{strReq("project_id", "project id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.ProjectPresence(reqStr(req, "project_id")))
		})
}

// addProjectUpdateTools registers a project's update feed: page through it,
// post one, rewrite one. Deleting an update stays a human's.
func addProjectUpdateTools(ts toolSet) {
	ts.add("project_updates", "One page of a project's updates, newest first: {updates, nextCursor}. Pass nextCursor back as before for the next page; nextCursor null is the end. node_id keeps the updates about that node and the nodes below it; pending keeps the proposals still waiting for your decision",
		[]mcp.ToolOption{strReq("project_id", "project id"), strOpt("before", "the nextCursor of the page before; omit for the newest"),
			mcp.WithNumber("limit", mcp.Description("updates per page, 1 to 100 (default 50)")),
			strOpt("node_id", "node id: only updates about this node and the nodes below it"),
			mcp.WithBoolean("pending", mcp.Description("only proposals with no decision yet, when you may decide them"))},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			limit, _ := req.GetArguments()["limit"].(float64)
			pending, _ := req.GetArguments()["pending"].(bool)
			return call(c.ProjectUpdates(reqStr(req, "project_id"), apiclient.UpdatesQuery{Before: optStr(req, "before"),
				Limit: int(limit), Node: optStr(req, "node_id"), Pending: pending}))
		})
	ts.add("update_add", "Post a progress, blocker or note update to a project",
		[]mcp.ToolOption{strReq("project_id", "project id"), strReq("kind", "progress|blocker|note"), strReq("content", "the update text")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.AddProjectUpdate(reqStr(req, "project_id"),
				apiclient.ProjectUpdate{Kind: reqStr(req, "kind"), Content: reqStr(req, "content")}))
		})
	ts.add("update_edit", "Rewrite the text of a project update (its author or a workspace admin)",
		[]mcp.ToolOption{strReq("update_id", "update id"), strReq("content", "the new text")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.EditProjectUpdate(reqStr(req, "update_id"), reqStr(req, "content")))
		})
}
