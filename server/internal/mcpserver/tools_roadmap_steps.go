package mcpserver

import (
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// addRoadmapStepsTools registers one level of the roadmap in dependency order,
// the Canvas's view (docs/architecture/roadmap-tree.md, "Steps").
func addRoadmapStepsTools(ts toolSet) {
	ts.add("roadmap_steps", "One level of a project's roadmap in order: the steps under root (default: your token's root, else the project's one root), each with its stage, its loop group, the steps it comes after with link counts, how many items it holds and what its work waits on outside the level. Open a step by passing its id as root",
		[]mcp.ToolOption{strReq("project_id", "project id"), strOpt("root", "node id: the steps under this node")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.RoadmapSteps(reqStr(req, "project_id"), optStr(req, "root")))
		})
}
