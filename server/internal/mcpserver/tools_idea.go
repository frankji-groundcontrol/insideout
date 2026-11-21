package mcpserver

import (
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// addIdeaTools registers capture, list, rewrite and conversion of the caller's
// own ideas. Deleting an idea stays a human's.
func addIdeaTools(ts toolSet) {
	ts.add("idea_create", "Capture a new idea in a workspace",
		[]mcp.ToolOption{strReq("workspace_id", "workspace id"), strReq("title", "idea title"), strOpt("content", "idea body")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.IdeaCreate(reqStr(req, "workspace_id"), reqStr(req, "title"), optStr(req, "content")))
		})
	ts.add("idea_list", "List your own ideas in a workspace, with full content (ideas are private to their author; dropped ideas are not listed)",
		[]mcp.ToolOption{strReq("workspace_id", "workspace id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.Ideas(reqStr(req, "workspace_id")))
		})
	ts.add("idea_update", "Rewrite an idea's title and, if given, its content (title required; omit content to keep the body, the word clear clears it)",
		[]mcp.ToolOption{strReq("idea_id", "idea id"), strReq("title", "new title"), strOpt("content", "idea body (omit to keep it; clear clears it)")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.UpdateIdea(reqStr(req, "idea_id"), reqStr(req, "title"), apiclient.PatchText(optStr(req, "content"))))
		})
	ts.add("idea_convert", "Turn an idea into a PRD and open the coach conversation. The coach speaks the locale you pass (en or zh): pass the language the idea is written in",
		[]mcp.ToolOption{strReq("idea_id", "idea id"), strOpt("locale", "language the coach speaks, en or zh (default: this server's LC_ALL, LC_MESSAGES or LANG, else en)")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			locale, err := apiclient.ConvertLocale(optStr(req, "locale"))
			if err != nil {
				return "", err
			}
			return call(c.IdeaConvert(reqStr(req, "idea_id"), locale))
		})
}
