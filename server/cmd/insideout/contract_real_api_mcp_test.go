package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/server"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/mcpserver"
	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
)

// The MCP half of the contract test (contract_real_api_test.go): the tool
// registry the MCP binary serves, called over JSON-RPC inside this process, with
// the real API behind it. A tool that sends a route or a field the API no longer
// has comes back as an error here; a canned server would never say so.

// kv is a tool's arguments.
type kv = map[string]any

// toolOut is what one tools/call answered.
type toolOut struct {
	text    string
	isError bool
}

// callTool sends one tools/call to the server the way an MCP client does.
func callTool(t *testing.T, s *server.MCPServer, name string, args kv) toolOut {
	t.Helper()
	req, err := json.Marshal(kv{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": kv{"name": name, "arguments": args}})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(s.HandleMessage(context.Background(), req))
	if err != nil {
		t.Fatal(err)
	}
	var res struct {
		Error  *struct{ Message string }
		Result struct {
			Content []struct{ Text string }
			IsError bool
		}
	}
	if err := json.Unmarshal(raw, &res); err != nil || res.Error != nil || len(res.Result.Content) != 1 {
		t.Fatalf("tools/call %s answered %s", name, raw)
	}
	return toolOut{res.Result.Content[0].Text, res.Result.IsError}
}

// toolData calls a tool that must succeed and returns what it put inside the
// <prd_data> fence that every product tool's success arrives in (AIDEV-08).
func toolData(t *testing.T, s *server.MCPServer, name string, args kv) string {
	t.Helper()
	out := callTool(t, s, name, args)
	if out.isError {
		t.Fatalf("%s failed: %s", name, out.text)
	}
	const open, shut = "\n<prd_data>", "</prd_data>" // the label above the fence names the tag too
	i := strings.Index(out.text, open)
	if i < 0 || !strings.HasSuffix(out.text, shut) {
		t.Fatalf("%s: the result is not a label above data inside a <prd_data> fence: %s", name, out.text)
	}
	return out.text[i+len(open) : len(out.text)-len(shut)]
}

func toolJSON[T any](t *testing.T, s *server.MCPServer, name string, args kv) T {
	t.Helper()
	return into[T](t, toolData(t, s, name, args))
}

// mcpTools runs the tools of an MCP process started after `insideout login`,
// one or more per family, on the rows the CLI phase made. The saved login is a
// JWT, so the tools run on an agent-scope token that the process trades it for
// at the real POST /agent/token, and every route they use must be one the
// server's agent scope lets through.
func (w *contractWorld) mcpTools(t *testing.T, r ids) {
	t.Cleanup(apiclient.ForgetAgentToken)
	s := mcpserver.Build(w.base)

	t.Run("read", func(t *testing.T) {
		if me := toolJSON[doc](t, s, "whoami", nil); me.Email != w.email {
			t.Fatalf("whoami = %+v, want %s", me, w.email)
		}
		wantListed(t, "workspaces", toolData(t, s, "workspaces", nil), r.workspace)
		wantListed(t, "projects", toolData(t, s, "projects", kv{"workspace_id": r.workspace}), r.project)
		wantListed(t, "roadmap_list", toolData(t, s, "roadmap_list", kv{"project_id": r.project}), r.node)
		if prd := toolJSON[doc](t, s, "prd", kv{"prd_id": r.prd}); prd.Sections["background"] != "Why now" {
			t.Fatalf("prd = %+v, want the background the CLI wrote", prd)
		}
		if revs := toolJSON[[]doc](t, s, "revisions", kv{"prd_id": r.prd}); len(revs) != 1 {
			t.Fatalf("revisions = %+v, want the CLI's one snapshot", revs)
		}
		for name, args := range map[string]kv{ // the rest answer without error
			"readiness": {"prd_id": r.prd}, "versions": {"prd_id": r.prd}, "view": {"prd_id": r.prd},
			"prd_export": {"prd_id": r.prd}, "progress": {"project_id": r.project},
			"presence": {"project_id": r.project}, "guide": {"project_id": r.project}, "roadmap_evidence": {"node_id": r.node},
		} {
			t.Run(name, func(t *testing.T) { toolData(t, s, name, args) })
		}
	})

	t.Run("write", func(t *testing.T) {
		idea := toolJSON[doc](t, s, "idea_create", kv{"workspace_id": r.workspace, "title": "Tool idea " + r.sfx, "content": "an idea from a tool"})
		if idea.ID == "" || idea.WorkspaceID != r.workspace {
			t.Fatalf("idea_create = %+v, want an idea of workspace %s", idea, r.workspace)
		}
		wantListed(t, "idea_list", toolData(t, s, "idea_list", kv{"workspace_id": r.workspace}), idea.ID)
		if edited := toolJSON[doc](t, s, "idea_update", kv{"idea_id": idea.ID, "title": "Edited " + r.sfx, "content": "edited"}); edited.Title != "Edited "+r.sfx || edited.Content != "edited" {
			t.Fatalf("idea_update = %+v, want the new title and content", edited)
		}
		if made := toolJSON[doc](t, s, "idea_convert", kv{"idea_id": idea.ID}); made.PrdID == "" || made.ConversationID == "" {
			t.Fatalf("idea_convert = %+v, want a PRD and its coach conversation", made)
		}
		ws := toolJSON[doc](t, s, "workspace_create", kv{"title": "Tool workspace " + r.sfx})
		if pr := toolJSON[doc](t, s, "project_create", kv{"workspace_id": ws.ID, "title": "Tool board " + r.sfx}); ws.ID == "" || pr.WorkspaceID != ws.ID {
			t.Fatalf("workspace_create %+v, project_create %+v, want a project of the new workspace", ws, pr)
		}

		child := toolJSON[doc](t, s, "roadmap_add", kv{"project_id": r.project, "title": "Child " + r.sfx, "parent_id": r.node})
		if child.ID == "" || child.ParentID != r.node {
			t.Fatalf("roadmap_add = %+v, want a child of node %s", child, r.node)
		}
		wantListed(t, "roadmap_list", toolData(t, s, "roadmap_list", kv{"project_id": r.project}), child.ID)
		if renamed := toolJSON[doc](t, s, "roadmap_update", kv{"node_id": child.ID, "title": "Renamed " + r.sfx}); renamed.Title != "Renamed "+r.sfx {
			t.Fatalf("roadmap_update = %+v, want the new title", renamed)
		}
		toolData(t, s, "roadmap_move", kv{"node_id": child.ID, "parent_id": "", "position": 0}) // to the root
		nodes := map[string]doc{}
		for _, n := range toolJSON[[]doc](t, s, "roadmap_list", kv{"project_id": r.project}) {
			nodes[n.ID] = n
		}
		if n, listed := nodes[child.ID]; !listed || n.ParentID != "" || n.Title != "Renamed "+r.sfx {
			t.Fatalf("node after roadmap_move = %+v (listed %v), want it at the root with its new title", n, listed)
		}

		toolData(t, s, "prd_update", kv{"prd_id": r.prd, "sections_json": `{"goals":"Ship it"}`})
		if prd := toolJSON[doc](t, s, "prd", kv{"prd_id": r.prd}); prd.Sections["goals"] != "Ship it" || prd.Sections["background"] != "Why now" {
			t.Fatalf("prd after prd_update = %+v, want the tool's goals beside the CLI's background", prd)
		}
		snap := toolJSON[doc](t, s, "snapshot", kv{"prd_id": r.prd, "note": "from a tool"})
		if revs := toolJSON[[]doc](t, s, "revisions", kv{"prd_id": r.prd}); len(revs) != 2 || revs[0].Revision != snap.Revision || snap.Revision <= revs[1].Revision {
			t.Fatalf("snapshot %+v, revisions %+v, want it listed first, after the CLI's", snap, revs)
		}
	})

	t.Run("agent", func(t *testing.T) {
		ctx := toolJSON[doc](t, s, "agent_context", kv{"project_id": r.project, "mode": "review"})
		if ctx.ProjectID != r.project || ctx.Mode != "review" || ctx.ProjectTitle != "Board "+r.sfx {
			t.Fatalf("agent_context = %+v, want the review context of project %s", ctx, r.project)
		}
		cp := toolJSON[doc](t, s, "checkpoint", kv{"project_id": r.project, "node_id": r.node, "summary": "Wired up by the contract test"})
		if cp.ID == "" || !cp.Recorded || cp.Kind != "agent_checkpoint" {
			t.Fatalf("checkpoint = %+v, want a recorded agent_checkpoint", cp)
		}
		prop := toolJSON[doc](t, s, "propose", kv{"project_id": r.project, "kind": "structure", "summary": "Split the step",
			"items": []kv{{"title": "Proposed " + r.sfx}}})
		if prop.ID == "" || !prop.Proposed || prop.Kind != "agent_proposal" {
			t.Fatalf("propose = %+v, want an open agent_proposal", prop)
		}

		// A human decides, on the CLI: the agent scope the tools run on cannot.
		decided := cliJSON[doc](t, "idea", "proposal-decide", "--accept", "--apply", prop.ID)
		if decided.Decision != "accepted" || len(decided.AppliedNodes) != 1 {
			t.Fatalf("proposal-decide = %+v, want it accepted, with the one proposed node applied", decided)
		}
		wantListed(t, "roadmap list", cliOut(t, "roadmap", "list", r.project), decided.AppliedNodes[0])
	})

	t.Run("draft", func(t *testing.T) {
		// This server has no roadmap planner, so the drafting tools say what the API says.
		for name, args := range map[string]kv{"build": {"prd_id": r.prd}, "expand": {"node_id": r.node}} {
			if out := callTool(t, s, name, args); !out.isError || !strings.Contains(out.text, "roadmap planner is not configured") {
				t.Fatalf("%s = %+v, want the API's planner-not-configured answer", name, out)
			}
		}
		built := toolJSON[doc](t, s, "build_confirm", kv{"prd_id": r.prd, "tree": `{"title":"Plan","children":[{"title":"One"},{"title":"Two"}]}`})
		if built.ProjectID == "" || built.NodeCount != 3 {
			t.Fatalf("build_confirm = %+v, want a project holding the 3 drafted nodes", built)
		}
		byTitle := map[string]doc{}
		for _, n := range toolJSON[[]doc](t, s, "roadmap_list", kv{"project_id": built.ProjectID}) {
			byTitle[n.Title] = n
		}
		plan, one := byTitle["Plan"], byTitle["One"]
		if len(byTitle) != 3 || plan.ID == "" || plan.ParentID != "" || one.ParentID != plan.ID || byTitle["Two"].ParentID != plan.ID {
			t.Fatalf("drafted roadmap = %+v, want Plan with children One and Two", byTitle)
		}
		subs := toolJSON[[]doc](t, s, "expand_confirm", kv{"node_id": one.ID, "children": `[{"title":"Sub A"}]`})
		if len(subs) != 1 || subs[0].Title != "Sub A" || subs[0].ParentID != one.ID {
			t.Fatalf("expand_confirm = %+v, want Sub A under One", subs)
		}
	})
}

// mcpSignedOut is an MCP process nobody has signed in. It offers every tool; the
// public signup-mode read works and the rest say to sign in; the auth tools
// open and close a session held in the process, which is then the login the
// other tools run on.
func (w *contractWorld) mcpSignedOut(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // no saved login to fall back on
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("INSIDEOUT_API", "")
	t.Setenv("INSIDEOUT_TOKEN", "")
	t.Cleanup(apiclient.ForgetAgentToken)
	s := mcpserver.Build(w.base)
	notSignedIn := func(when string) {
		t.Helper()
		if out := callTool(t, s, "whoami", nil); !out.isError || !strings.Contains(out.text, "not signed in") {
			t.Fatalf("whoami %s = %+v, want the error that says to sign in", when, out)
		}
	}
	notSignedIn("with no login")

	testlock.SignupMode(t, w.st.Pool) // the mode is a global row other packages' tests flip
	if got, want := toolData(t, s, "signup_mode", nil), w.st.SignupMode(context.Background()); got != want {
		t.Fatalf("signup_mode = %q, want what the database holds (%q)", got, want)
	}

	t.Cleanup(func() { callTool(t, s, "auth_logout", kv{"confirm": true}) }) // the session is process state: leave none behind
	if out := callTool(t, s, "auth_login", kv{"email": w.email, "password": w.password}); out.isError || !strings.Contains(out.text, "signed in as "+w.email) {
		t.Fatalf("auth_login = %+v, want it to sign %s in", out, w.email)
	}
	if me := toolJSON[doc](t, s, "whoami", nil); me.Email != w.email {
		t.Fatalf("whoami after auth_login = %+v, want %s", me, w.email)
	}
	if out := callTool(t, s, "auth_logout", kv{"confirm": true}); out.isError || out.text != "session cleared" {
		t.Fatalf("auth_logout = %+v, want the session cleared", out)
	}
	notSignedIn("after auth_logout")
}
