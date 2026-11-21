package mcpserver

import (
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// addProjectTools registers the project and PRD reads an agent token may make:
// the paged updates feed and a workspace's PRD list (rv5-PARB-07, PARB-03).
// Reading, editing or deleting one project, posting an update and editing a
// workspace run on routes the server keeps for a human, so they are CLI and
// web verbs with no tool here.
func addProjectTools(
	addTextTool func(name, desc string, opts []mcp.ToolOption, call func(c *apiclient.Client, req mcp.CallToolRequest) (string, error)),
	strReq, strOpt func(name, desc string) mcp.ToolOption,
) {
	addTextTool("project_updates", "One page of a project's updates, newest first: {updates, nextCursor}. Pass nextCursor back as before for the next page; nextCursor null is the end",
		[]mcp.ToolOption{strReq("project_id", "project id"), strOpt("before", "the nextCursor of the page before; omit for the newest"),
			mcp.WithNumber("limit", mcp.Description("updates per page, 1 to 100 (default 50)"))},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			limit, _ := req.GetArguments()["limit"].(float64)
			return call(c.ProjectUpdates(reqStr(req, "project_id"), optStr(req, "before"), int(limit)))
		})
	addTextTool("prd_list", "List every PRD of a workspace, newest first: id, title, status, author, and the project it built (null until built)",
		[]mcp.ToolOption{strReq("workspace_id", "workspace id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.Prds(reqStr(req, "workspace_id")))
		})
}
