package mcpserver

import (
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// addWorkspaceTools registers the workspace list, read, create, join and edit.
// Deleting a workspace stays a human's.
func addWorkspaceTools(ts toolSet) {
	ts.add("workspaces", "List workspaces visible to the user", nil,
		func(c *apiclient.Client, _ mcp.CallToolRequest) (string, error) { return call(c.Workspaces()) })
	ts.add("workspace", "Get one workspace", []mcp.ToolOption{strReq("workspace_id", "workspace id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.Workspace(reqStr(req, "workspace_id")))
		})
	ts.add("workspace_create", "Create a workspace (creator becomes owner)",
		[]mcp.ToolOption{strReq("title", "workspace title"), strOpt("description", "what this workspace is for")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.CreateWorkspace(reqStr(req, "title"), optStr(req, "description")))
		})
	ts.add("workspace_join", "Join a workspace by its join code",
		[]mcp.ToolOption{strReq("code", "workspace join code")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.JoinWorkspace(reqStr(req, "code")))
		})
	ts.add("workspace_update", "Edit a workspace's title, description or cover image; a field left out keeps its value, the word clear clears the description or cover",
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
}
