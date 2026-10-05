package mcpserver

// The product tools — one per CLI verb, names/arguments/output 1:1 with
// the CLI (docs/plans/2026-08-21-cli-mcp-parity.md). Reads return the
// API's raw JSON so surfaces cannot drift from the truth.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

func addProductTools(s *server.MCPServer, api string) {
	addTextTool := func(name, desc string, opts []mcp.ToolOption, call func(c *apiclient.Client, req mcp.CallToolRequest) (string, error)) {
		all := append([]mcp.ToolOption{mcp.WithDescription(desc)}, opts...)
		s.AddTool(mcp.NewTool(name, all...), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			out, err := call(client(api), req)
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
			strOpt("description", "new description"), strOpt("status", "locked|pending|in_progress|done"),
			strOpt("deadline", "RFC3339 deadline, or empty string to clear")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			arg := func(k string) *string {
				if v, ok := req.GetArguments()[k].(string); ok && v != "" {
					return &v
				}
				return nil
			}
			deadline, hasDeadline := req.GetArguments()["deadline"].(string)
			var dl *string
			if hasDeadline {
				dl = &deadline
			}
			return call(c.RoadmapUpdate(reqStr(req, "node_id"), arg("title"), arg("description"), arg("status"), dl))
		})
	addTextTool("roadmap_move", "Re-parent / reposition a node",
		[]mcp.ToolOption{strReq("node_id", "node id"), strOpt("parent_id", "new parent id; empty = root"),
			mcp.WithNumber("position", mcp.Description("position among siblings (0-based)"))},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			var parent *string
			if v, ok := req.GetArguments()["parent_id"].(string); ok && v != "" {
				parent = &v
			}
			var pos *int
			if v, ok := req.GetArguments()["position"].(float64); ok {
				p := int(v)
				pos = &p
			}
			return call(c.RoadmapMove(reqStr(req, "node_id"), parent, pos))
		})
	addTextTool("roadmap_delete", "Delete a node", []mcp.ToolOption{strReq("node_id", "node id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			if err := c.RoadmapDelete(reqStr(req, "node_id")); err != nil {
				return "", err
			}
			return "deleted", nil
		})
	addDraftTools(addTextTool, strReq)
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
	addTextTool("revisions", "List a PRD's revision snapshots",
		[]mcp.ToolOption{strReq("prd_id", "PRD id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.PrdRevisions(reqStr(req, "prd_id")))
		})
	addTextTool("snapshot", "Record a revision snapshot of the working PRD (working version, not a Commit)",
		[]mcp.ToolOption{strReq("prd_id", "PRD id"), strOpt("note", "optional note")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			note, _ := req.GetArguments()["note"].(string)
			return call(c.SnapshotPrd(reqStr(req, "prd_id"), note))
		})
	addTextTool("progress", "Time-first Progress view: Now (deadlined work only), at most three Next, Done count, plus in-progress items missing a deadline",
		[]mcp.ToolOption{strReq("project_id", "project id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.RoadmapProgress(reqStr(req, "project_id")))
		})
	addTextTool("presence", "Who is currently viewing a project (canvas presence snapshot)",
		[]mcp.ToolOption{strReq("project_id", "project id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.ProjectPresence(reqStr(req, "project_id")))
		})
	addTextTool("idea_create", "Capture a new idea in a workspace",
		[]mcp.ToolOption{strReq("workspace_id", "workspace id"), strReq("title", "idea title"), strOpt("content", "idea body")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			content, _ := req.GetArguments()["content"].(string)
			return call(c.IdeaCreate(reqStr(req, "workspace_id"), reqStr(req, "title"), content))
		})
	addTextTool("idea_convert", "Turn an idea into a PRD and open the coach conversation",
		[]mcp.ToolOption{strReq("idea_id", "idea id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.IdeaConvert(reqStr(req, "idea_id")))
		})
	addTextTool("proposal_decide", "Human accept/reject of an agent proposal (owner or admin; the Decision Log entry)",
		[]mcp.ToolOption{strReq("update_id", "proposal update id"), strReq("decision", "accepted|rejected"), strOpt("reason", "why"), mcp.WithBoolean("apply", mcp.Description("apply the proposal structured items on acceptance"))},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			reason, _ := req.GetArguments()["reason"].(string)
			apply, _ := req.GetArguments()["apply"].(bool)
			return call(c.DecideProposal(reqStr(req, "update_id"), reqStr(req, "decision"), reason, apply))
		})
	addTextTool("agent_context", "Compact, focus-scoped agent context for a project (mode brainstorming|implementation|review)",
		[]mcp.ToolOption{strReq("project_id", "project id"), strOpt("mode", "brainstorming|implementation|review"), strOpt("focus", "roadmap node id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			mode, _ := req.GetArguments()["mode"].(string)
			focus, _ := req.GetArguments()["focus"].(string)
			return call(c.AgentContext(reqStr(req, "project_id"), mode, focus))
		})
	addTextTool("checkpoint", "Record agent work done on a project (never applies changes)",
		[]mcp.ToolOption{strReq("project_id", "project id"), strReq("summary", "what was done"), strOpt("node_id", "roadmap node id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			node, _ := req.GetArguments()["node_id"].(string)
			return call(c.AgentCheckpoint(reqStr(req, "project_id"), node, reqStr(req, "summary")))
		})
	addTextTool("propose", "Propose a structure/scope/priority change; a human decides",
		[]mcp.ToolOption{strReq("project_id", "project id"), strReq("kind", "structure|scope|priority"),
			strReq("summary", "the proposal"), strOpt("detail", "extra detail")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			detail, _ := req.GetArguments()["detail"].(string)
			var items []string
			if arr, ok := req.GetArguments()["items"].([]any); ok {
				for _, v := range arr {
					if m, ok := v.(map[string]any); ok {
						title, _ := m["title"].(string)
						hint, _ := m["parentHint"].(string)
						if title != "" {
							items = append(items, title+"@"+hint)
						}
					}
				}
			}
			return call(c.AgentPropose(reqStr(req, "project_id"), reqStr(req, "kind"), reqStr(req, "summary"), detail, items))
		})
	addTextTool("view", "Audience projection of a PRD (decision|management|delivery|validation): ordered sections with whys, readiness gaps, latest commit",
		[]mcp.ToolOption{strReq("prd_id", "PRD id"), strOpt("audience", "decision|management|delivery|validation (default decision)")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			aud, _ := req.GetArguments()["audience"].(string)
			if aud == "" {
				aud = "decision"
			}
			return call(c.PrdView(reqStr(req, "prd_id"), aud))
		})
	addTextTool("invite_create", "Admin: mint an email-bound invitation code (response contains the one-time code and signup URL)",
		[]mcp.ToolOption{strReq("email", "invitee email"), strOpt("ttl_hours", "lifetime in hours (default 168)")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			ttl := 0
			if v, ok := req.GetArguments()["ttl_hours"].(float64); ok {
				ttl = int(v)
			}
			return call(c.CreateInvitation(reqStr(req, "email"), ttl))
		})
	addTextTool("invite_list", "Admin: list invitations (email, expiry, used/revoked state — no codes)",
		nil,
		func(c *apiclient.Client, _ mcp.CallToolRequest) (string, error) {
			return call(c.ListInvitations())
		})
	addTextTool("idea_list", "List a workspace's idea inbox with full content",
		[]mcp.ToolOption{strReq("workspace_id", "workspace id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.Ideas(reqStr(req, "workspace_id")))
		})
	addTextTool("idea_update", "Rewrite an idea's title and content (title required)",
		[]mcp.ToolOption{strReq("idea_id", "idea id"), strReq("title", "new title"), strOpt("content", "idea body")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			content, _ := req.GetArguments()["content"].(string)
			return call(c.UpdateIdea(reqStr(req, "idea_id"), reqStr(req, "title"), content))
		})
	addTextTool("idea_delete", "Drop an idea (soft: it stays in idea_list with status \"dropped\"; filter client-side)",
		[]mcp.ToolOption{strReq("idea_id", "idea id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			if err := c.DeleteIdea(reqStr(req, "idea_id")); err != nil {
				return "", err
			}
			return "deleted", nil
		})
	addTextTool("prd_update", "Edit PRD sections/title directly (section keys: background, users, goals, nonGoals, stories, requirements, constraints, risks); optional status transition applies after edits",
		[]mcp.ToolOption{strReq("prd_id", "PRD id"), strOpt("title", "new title"),
			strOpt("status", "status transition after edits"), strOpt("sections_json", "JSON object of section key → text")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			var title *string
			if v, ok := req.GetArguments()["title"].(string); ok && v != "" {
				title = &v
			}
			sections := map[string]string{}
			if raw, ok := req.GetArguments()["sections_json"].(string); ok && raw != "" {
				if err := json.Unmarshal([]byte(raw), &sections); err != nil {
					return "", fmt.Errorf("sections_json must be a JSON object: %w", err)
				}
			}
			out, err := c.UpdatePrd(reqStr(req, "prd_id"), title, sections)
			if err != nil {
				return "", err
			}
			if status, ok := req.GetArguments()["status"].(string); ok && status != "" {
				return call(c.SetPrdStatus(reqStr(req, "prd_id"), status))
			}
			return string(out), nil
		})
	addTextTool("roadmap_evidence", "Read the GitHub evidence attached to a roadmap node (the read half of bind → sync → checkpoint)",
		[]mcp.ToolOption{strReq("node_id", "roadmap node id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.NodeEvidence(reqStr(req, "node_id")))
		})
	addTextTool("workspace_create", "Create a workspace (creator becomes owner)",
		[]mcp.ToolOption{strReq("title", "workspace title"), strOpt("description", "what this workspace is for")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			desc, _ := req.GetArguments()["description"].(string)
			return call(c.CreateWorkspace(reqStr(req, "title"), desc))
		})
	addTextTool("workspace_join", "Join a workspace by its join code",
		[]mcp.ToolOption{strReq("code", "workspace join code")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.JoinWorkspace(reqStr(req, "code")))
		})
	addTextTool("prd_export", "Export a PRD as text (MCP has no filesystem — the content is the result): markdown by default, or print HTML; an optional audience projects the sections first",
		[]mcp.ToolOption{strReq("prd_id", "PRD id"), strOpt("format", "markdown|print (default markdown)"),
			strOpt("audience", "decision|management|delivery|validation (markdown only)")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			format, _ := req.GetArguments()["format"].(string)
			aud, _ := req.GetArguments()["audience"].(string)
			raw, err := c.PrdExport(reqStr(req, "prd_id"), format, aud)
			return string(raw), err
		})
	addTextTool("project_create", "Create a project inside a workspace",
		[]mcp.ToolOption{strReq("workspace_id", "workspace id"), strReq("title", "project title"), strOpt("description", "project description")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			desc, _ := req.GetArguments()["description"].(string)
			return call(c.CreateProject(reqStr(req, "workspace_id"), reqStr(req, "title"), desc))
		})
	addTextTool("versions", "List a PRD's committed versions, newest first, with diffs",
		[]mcp.ToolOption{strReq("prd_id", "PRD id")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			return call(c.PrdVersions(reqStr(req, "prd_id")))
		})
}
