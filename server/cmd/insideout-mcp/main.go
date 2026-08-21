// Command insideout-mcp is the MCP projection of the InsideOut product:
// one tool per CLI verb over the same /api/v1 contract
// (docs/plans/2026-08-21-cli-mcp-parity.md — names, arguments, and
// output stay 1:1 with the CLI). Configure with INSIDEOUT_API (default:
// hosted API) and INSIDEOUT_TOKEN; there is no login tool because the
// token is environment state, not a conversation.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

const defaultAPIBase = "https://server-production-9c338.up.railway.app/api/v1"

func main() {
	api := flag.String("api", envOr("INSIDEOUT_API", defaultAPIBase), "API base URL")
	flag.Parse()

	s := server.NewMCPServer("insideout", "0.1.0",
		server.WithToolCapabilities(false),
		server.WithInstructions("InsideOut: ideas → PRDs → roadmaps. `guide` scaffolds the "+
			"repo-side insideout.yaml matching guide; roadmap_* manage branches; build/expand "+
			"run the agent planner. Auth: INSIDEOUT_TOKEN env."))

	addTextTool := func(name, desc string, opts []mcp.ToolOption, call func(c *apiclient.Client, req mcp.CallToolRequest) (string, error)) {
		all := append([]mcp.ToolOption{mcp.WithDescription(desc)}, opts...)
		s.AddTool(mcp.NewTool(name, all...), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			out, err := call(client(*api), req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(out), nil
		})
	}

	strOpt := func(name, desc string) mcp.ToolOption {
		return mcp.WithString(name, mcp.Description(desc))
	}
	strReq := func(name, desc string) mcp.ToolOption {
		return mcp.WithString(name, mcp.Required(), mcp.Description(desc))
	}

	addTextTool("whoami", "GET /me — the authenticated user", nil,
		func(c *apiclient.Client, _ mcp.CallToolRequest) (string, error) { return call(c.Whoami()) })
	addTextTool("workspaces", "List workspaces visible to the user", nil,
		func(c *apiclient.Client, _ mcp.CallToolRequest) (string, error) { return call(c.Workspaces()) })
	addTextTool("projects", "List projects in a workspace", []mcp.ToolOption{strReq("workspace_id", "workspace id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.Projects(reqStr(req, "workspace_id")))
		})
	addTextTool("prd", "Get one PRD", []mcp.ToolOption{strReq("prd_id", "PRD id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.Prd(reqStr(req, "prd_id")))
		})
	addTextTool("roadmap_list", "List a project's roadmap nodes", []mcp.ToolOption{strReq("project_id", "project id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.RoadmapList(reqStr(req, "project_id")))
		})
	addTextTool("roadmap_add", "Create a roadmap node (branch or leaf)",
		[]mcp.ToolOption{strReq("project_id", "project id"), strReq("title", "node title"),
			strOpt("description", "node description"), strOpt("parent_id", "parent node id; empty = root")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			var parent *string
			if v, ok := req.GetArguments()["parent_id"].(string); ok && v != "" {
				parent = &v
			}
			desc, _ := req.GetArguments()["description"].(string)
			return call(c.RoadmapAdd(reqStr(req, "project_id"), reqStr(req, "title"), desc, parent))
		})
	addTextTool("roadmap_update", "Partially update a node (title/description/status)",
		[]mcp.ToolOption{strReq("node_id", "node id"), strOpt("title", "new title"),
			strOpt("description", "new description"), strOpt("status", "locked|pending|in_progress|done")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			arg := func(k string) *string {
				if v, ok := req.GetArguments()[k].(string); ok && v != "" {
					return &v
				}
				return nil
			}
			return call(c.RoadmapUpdate(reqStr(req, "node_id"), arg("title"), arg("description"), arg("status")))
		})
	addTextTool("roadmap_move", "Re-parent / reposition a node",
		[]mcp.ToolOption{strReq("node_id", "node id"), strOpt("parent_id", "new parent id; empty = root")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			var parent *string
			if v, ok := req.GetArguments()["parent_id"].(string); ok && v != "" {
				parent = &v
			}
			return call(c.RoadmapMove(reqStr(req, "node_id"), parent, nil))
		})
	addTextTool("roadmap_delete", "Delete a node", []mcp.ToolOption{strReq("node_id", "node id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			if err := c.RoadmapDelete(reqStr(req, "node_id")); err != nil {
				return "", err
			}
			return "deleted", nil
		})
	addTextTool("build", "Agent: PRD → project with a branched roadmap",
		[]mcp.ToolOption{strReq("prd_id", "PRD id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.BuildFromPrd(reqStr(req, "prd_id"), 0))
		})
	addTextTool("expand", "Agent: grow one node into subtasks", []mcp.ToolOption{strReq("node_id", "node id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.ExpandNode(reqStr(req, "node_id")))
		})
	addTextTool("guide", "Scaffold the repo-side insideout.yaml matching guide for a project's roadmap; commit it at the repo root",
		[]mcp.ToolOption{strReq("project_id", "project id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			g, err := c.Guide(reqStr(req, "project_id"))
			return string(g), err
		})
	addTextTool("repo_set", "Bind a GitHub repo to a project",
		[]mcp.ToolOption{strReq("project_id", "project id"), strReq("repo_url", "https://github.com/owner/repo")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.SetRepo(reqStr(req, "project_id"), reqStr(req, "repo_url")))
		})
	addTextTool("sync", "Pull GitHub evidence for a project now", []mcp.ToolOption{strReq("project_id", "project id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.SyncGithub(reqStr(req, "project_id")))
		})
	addTextTool("commit", "Human Commit: freeze the working PRD as an immutable version (name, audience, summary, unresolved items, diff)",
		[]mcp.ToolOption{strReq("prd_id", "PRD id"), strReq("name", "version name"),
			strReq("audience", "decision|management|delivery|validation"),
			strOpt("summary", "change summary"), strOpt("note", "decision log note")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			var unresolved []string
			if arr, ok := req.GetArguments()["unresolved"].([]any); ok {
				for _, v := range arr {
					if sv, ok := v.(string); ok {
						unresolved = append(unresolved, sv)
					}
				}
			}
			summary, _ := req.GetArguments()["summary"].(string)
			note, _ := req.GetArguments()["note"].(string)
			return call(c.CommitPrd(reqStr(req, "prd_id"), reqStr(req, "name"), reqStr(req, "audience"), summary, unresolved, note))
		})
	addTextTool("readiness", "Per-audience gap disclosure for 'form a version now': what is missing for each audience, why, and the suggested unresolved items to carry into a Commit",
		[]mcp.ToolOption{strReq("prd_id", "PRD id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.PrdReadiness(reqStr(req, "prd_id")))
		})
	addTextTool("versions", "List a PRD's committed versions, newest first, with diffs",
		[]mcp.ToolOption{strReq("prd_id", "PRD id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.PrdVersions(reqStr(req, "prd_id")))
		})

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintln(os.Stderr, "insideout-mcp:", err)
		os.Exit(1)
	}
}

// reqStr reads a required string argument; MCP validation already
// guarantees presence, so a miss degrades to "" and the API answers 4xx.
func reqStr(req mcp.CallToolRequest, name string) string {
	v, _ := req.RequireString(name)
	return v
}

func client(api string) *apiclient.Client {
	c := apiclient.New(api)
	c.SetToken(os.Getenv("INSIDEOUT_TOKEN"))
	return c
}

func call(raw json.RawMessage, err error) (string, error) {
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
