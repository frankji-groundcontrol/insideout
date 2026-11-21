package mcpserver

import (
	"errors"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// errNothingToChange answers an edit tool given no field to change.
var errNothingToChange = errors.New("nothing to change: pass at least one field (the word clear clears one)")

// addProjectTools registers the project, workspace and PRD-list tools
// (rv5-PARB-04, PARA-06, PARB-07, PARB-03). The edits fill the CLI's request
// structs by its rule: an absent or empty value leaves a field, the word clear
// clears it. project_update takes no owner (an agent token may not change an
// owner), and deleting a project, a workspace or an update stays a human's.
func addProjectTools(
	addTextTool func(name, desc string, opts []mcp.ToolOption, call func(c *apiclient.Client, req mcp.CallToolRequest) (string, error)),
	strReq, strOpt func(name, desc string) mcp.ToolOption,
) {
	addTextTool("project", "Get one project with its newest page of updates and their nextCursor",
		[]mcp.ToolOption{strReq("project_id", "project id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.Project(reqStr(req, "project_id")))
		})
	addTextTool("project_update", "Edit a project's title, description or status; a field left out keeps its value, the word clear clears the description",
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
	addTextTool("project_updates", "One page of a project's updates, newest first: {updates, nextCursor}. Pass nextCursor back as before for the next page; nextCursor null is the end",
		[]mcp.ToolOption{strReq("project_id", "project id"), strOpt("before", "the nextCursor of the page before; omit for the newest"),
			mcp.WithNumber("limit", mcp.Description("updates per page, 1 to 100 (default 50)"))},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			limit, _ := req.GetArguments()["limit"].(float64)
			return call(c.ProjectUpdates(reqStr(req, "project_id"), optStr(req, "before"), int(limit)))
		})
	addTextTool("update_add", "Post a progress, blocker or note update to a project",
		[]mcp.ToolOption{strReq("project_id", "project id"), strReq("kind", "progress|blocker|note"), strReq("content", "the update text")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.AddProjectUpdate(reqStr(req, "project_id"),
				apiclient.ProjectUpdate{Kind: reqStr(req, "kind"), Content: reqStr(req, "content")}))
		})
	addTextTool("update_edit", "Rewrite the text of a project update (its author or a workspace admin)",
		[]mcp.ToolOption{strReq("update_id", "update id"), strReq("content", "the new text")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.EditProjectUpdate(reqStr(req, "update_id"), reqStr(req, "content")))
		})
	addTextTool("workspace", "Get one workspace", []mcp.ToolOption{strReq("workspace_id", "workspace id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.Workspace(reqStr(req, "workspace_id")))
		})
	addTextTool("workspace_update", "Edit a workspace's title, description or cover image; a field left out keeps its value, the word clear clears the description or cover",
		[]mcp.ToolOption{strReq("workspace_id", "workspace id"), strOpt("title", "new title"),
			strOpt("description", "new description, or clear"), strOpt("cover_url", "cover image URL, or clear")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			patch := apiclient.WorkspacePatch{Title: apiclient.NonEmpty(optStr(req, "title")),
				Description: apiclient.PatchText(optStr(req, "description")), CoverURL: apiclient.PatchText(optStr(req, "cover_url"))}
			if patch == (apiclient.WorkspacePatch{}) {
				return "", errNothingToChange
			}
			return call(patch.Apply(c, reqStr(req, "workspace_id")))
		})
	addTextTool("prd_list", "List every PRD of a workspace, newest first: id, title, status, author, and the project it built (null until built)",
		[]mcp.ToolOption{strReq("workspace_id", "workspace id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.Prds(reqStr(req, "workspace_id")))
		})
}
