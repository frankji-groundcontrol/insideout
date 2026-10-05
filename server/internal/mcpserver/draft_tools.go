package mcpserver

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// addDraftTools registers the roadmap draft and confirm verbs. A draft call
// returns the plan and writes nothing; the matching confirm call persists it.
func addDraftTools(
	addTextTool func(name, desc string, opts []mcp.ToolOption, call func(c *apiclient.Client, req mcp.CallToolRequest) (string, error)),
	strReq func(name, desc string) mcp.ToolOption,
) {
	addTextTool("build", "Agent: draft a branched roadmap from a PRD. Writes nothing until build_confirm.",
		[]mcp.ToolOption{strReq("prd_id", "PRD id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.BuildFromPrd(reqStr(req, "prd_id")))
		})
	addTextTool("build_confirm", "Agent: write a previously drafted roadmap tree",
		[]mcp.ToolOption{strReq("prd_id", "PRD id"), strReq("tree", "the drafted tree JSON from build"),
			mcp.WithNumber("expected_count", mcp.Description("live node count; required to confirm a replace_conflict 409 retry"))},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			count := 0
			if v, ok := req.GetArguments()["expected_count"].(float64); ok {
				count = int(v)
			}
			tree := json.RawMessage(reqStr(req, "tree"))
			if !json.Valid(tree) {
				return "", fmt.Errorf("tree must be the drafted JSON object from build")
			}
			return call(c.ConfirmBuildFromPrd(reqStr(req, "prd_id"), tree, count))
		})
	addTextTool("expand", "Agent: draft subtasks for one node. Writes nothing until expand_confirm.",
		[]mcp.ToolOption{strReq("node_id", "node id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.ExpandNode(reqStr(req, "node_id")))
		})
	addTextTool("expand_confirm", "Agent: write previously drafted subtasks",
		[]mcp.ToolOption{strReq("node_id", "node id"), strReq("children", "the drafted children JSON from expand")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			children := json.RawMessage(reqStr(req, "children"))
			if !json.Valid(children) {
				return "", fmt.Errorf("children must be the drafted JSON array from expand")
			}
			return call(c.ConfirmExpandNode(reqStr(req, "node_id"), children))
		})
}
