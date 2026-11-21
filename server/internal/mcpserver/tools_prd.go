package mcpserver

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// addPrdTools registers reading, listing, editing, projecting and exporting a
// PRD. Commit stays a human's.
func addPrdTools(ts toolSet) {
	ts.add("prd", "Get one PRD", []mcp.ToolOption{strReq("prd_id", "PRD id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.Prd(reqStr(req, "prd_id")))
		})
	ts.add("prd_list", "List every PRD of a workspace, newest first: id, title, status, author, and the project it built (null until built)",
		[]mcp.ToolOption{strReq("workspace_id", "workspace id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.Prds(reqStr(req, "workspace_id")))
		})
	ts.add("prd_update", "Edit PRD sections/title directly (section keys: background, users, goals, nonGoals, stories, requirements, constraints, risks); optional status transition applies after edits",
		[]mcp.ToolOption{strReq("prd_id", "PRD id"), strOpt("title", "new title"),
			strOpt("status", "status transition after edits"), strOpt("sections_json", "JSON object of section key → text")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			sections := map[string]string{}
			if raw := optStr(req, "sections_json"); raw != "" {
				if err := json.Unmarshal([]byte(raw), &sections); err != nil {
					return "", fmt.Errorf("sections_json must be a JSON object: %w", err)
				}
			}
			out, err := c.UpdatePrd(reqStr(req, "prd_id"), apiclient.NonEmpty(optStr(req, "title")), sections)
			if err != nil {
				return "", err
			}
			if status := optStr(req, "status"); status != "" {
				return call(c.SetPrdStatus(reqStr(req, "prd_id"), status))
			}
			return string(out), nil
		})
	ts.add("view", "Audience projection of a PRD (decision|management|delivery|validation): ordered sections with whys, readiness gaps, latest commit",
		[]mcp.ToolOption{strReq("prd_id", "PRD id"), strOpt("audience", "decision|management|delivery|validation (default decision)")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			aud := optStr(req, "audience")
			if aud == "" {
				aud = "decision"
			}
			return call(c.PrdView(reqStr(req, "prd_id"), aud))
		})
	ts.add("prd_export", "Export a PRD as text (MCP has no filesystem). The file content is the text between the <prd_data> tags: markdown by default, or print HTML; an optional audience projects the sections first",
		[]mcp.ToolOption{strReq("prd_id", "PRD id"), strOpt("format", "markdown|print (default markdown)"),
			strOpt("audience", "decision|management|delivery|validation (markdown only)")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			raw, err := c.PrdExport(reqStr(req, "prd_id"), optStr(req, "format"), optStr(req, "audience"))
			return string(raw), err
		})
}

// addPrdHistoryTools registers a PRD's readiness, revision snapshots and
// committed versions.
func addPrdHistoryTools(ts toolSet) {
	ts.add("readiness", "Per-audience gap disclosure for 'form a version now': what is missing for each audience, why, and the suggested unresolved items to carry into a Commit",
		[]mcp.ToolOption{strReq("prd_id", "PRD id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.PrdReadiness(reqStr(req, "prd_id")))
		})
	ts.add("revisions", "List a PRD's revision snapshots",
		[]mcp.ToolOption{strReq("prd_id", "PRD id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.PrdRevisions(reqStr(req, "prd_id")))
		})
	ts.add("snapshot", "Record a revision snapshot of the working PRD (working version, not a Commit)",
		[]mcp.ToolOption{strReq("prd_id", "PRD id"), strOpt("note", "optional note")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.SnapshotPrd(reqStr(req, "prd_id"), optStr(req, "note")))
		})
	ts.add("versions", "List a PRD's committed versions, newest first, with diffs",
		[]mcp.ToolOption{strReq("prd_id", "PRD id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.PrdVersions(reqStr(req, "prd_id")))
		})
}
