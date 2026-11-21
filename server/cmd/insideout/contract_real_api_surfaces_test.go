package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
	"github.com/frankji-groundcontrol/insideout/server/internal/mcpserver"
)

// rv5-PARB-04, PARB-03, PARB-07: the project, updates, workspace and PRD-list
// verbs on the real API, from the CLI and the MCP tools. An edit keeps what it
// was not told to change, and the updates feed pages to its end through the
// cursor. DATABASE_URL-gated.

// updatesPage is one page of GET /projects/{pid}/updates, or a project view
// with its first page embedded.
type updatesPage struct {
	ID, Title, Description, Status string
	PrdID                          *string
	Updates                        []doc
	NextCursor                     *string
}

func TestCLIAndMCPProjectVerbsOnTheRealAPI(t *testing.T) {
	w := newContractWorld(t)
	c := apiclient.New(w.base)
	if err := c.Login(w.email, w.password); err != nil {
		t.Fatalf("login: %v", err)
	}
	writeClientConfig(t, clientconfig.Config{APIBase: w.base, Token: c.Token(), RefreshToken: c.RefreshToken()})

	sfx := uuid.NewString()[:8]
	ws := cliJSON[doc](t, "workspaces", "create", "--title", "Surfaces "+sfx, "--desc", "kept")
	pr := cliJSON[doc](t, "projects", "create", "--title", "Board "+sfx, "--desc", "first", ws.ID)

	if got := cliJSON[updatesPage](t, "project", pr.ID); got.ID != pr.ID || got.Title != "Board "+sfx || got.Description != "first" || got.PrdID != nil {
		t.Fatalf("project = %+v, want the project made by hand, with no PRD", got)
	}
	if got := cliJSON[updatesPage](t, "project", "update", "--status", "paused", pr.ID); got.Status != "paused" || got.Title != "Board "+sfx || got.Description != "first" {
		t.Fatalf("project update --status = %+v, want paused with its title and description kept", got)
	}
	if got := cliJSON[updatesPage](t, "project", "update", "--description", "clear", pr.ID); got.Description != "" || got.Status != "paused" {
		t.Fatalf("project update --description clear = %+v, want no description and the status kept", got)
	}
	if got := cliJSON[doc](t, "workspace", "update", "--title", "Renamed "+sfx, ws.ID); got.Title != "Renamed "+sfx {
		t.Fatalf("workspace update = %+v, want the new title", got)
	}
	if got := cliJSON[updatesPage](t, "workspace", ws.ID); got.Description != "kept" {
		t.Fatalf("workspace = %+v, want the description an edit of the title left alone", got)
	}

	for i := range 3 {
		if u := cliJSON[doc](t, "update", "add", "--kind", "note", "--content", "note "+string(rune('a'+i)), pr.ID); u.Kind != "note" {
			t.Fatalf("update add = %+v, want a note", u)
		}
	}
	first := cliJSON[updatesPage](t, "updates", "--limit", "2", pr.ID)
	if len(first.Updates) != 2 || first.NextCursor == nil || first.Updates[0].Content != "note c" {
		t.Fatalf("updates --limit 2 = %+v, want the two newest notes and a cursor", first)
	}
	rest := cliJSON[updatesPage](t, "updates", "--before", *first.NextCursor, "--limit", "2", pr.ID)
	if len(rest.Updates) != 1 || rest.NextCursor != nil || rest.Updates[0].Content != "note a" {
		t.Fatalf("updates --before = %+v, want the oldest note and no cursor", rest)
	}

	idea := cliJSON[doc](t, "idea", "create", "--title", "Idea "+sfx, ws.ID)
	made := cliJSON[doc](t, "idea", "convert", idea.ID)
	wantListed(t, "prds", cliOut(t, "prds", ws.ID), made.PrdID)

	// rv5-PARB-06: the whole PRD, as markdown on stdout or print HTML in a file.
	if md := cliOut(t, "export", made.PrdID); !strings.Contains(md, "Idea "+sfx) {
		t.Fatalf("export printed %q, want the PRD's markdown with its title", md)
	}
	page := filepath.Join(t.TempDir(), "prd.html")
	cliOut(t, "export", "--format", "print", "--out", page, made.PrdID)
	if html, err := os.ReadFile(page); err != nil || !strings.HasPrefix(string(html), "<!DOCTYPE html>") {
		t.Fatalf("export --format print --out wrote %.80q (%v), want the print HTML", html, err)
	}

	t.Cleanup(apiclient.ForgetAgentToken)
	s := mcpserver.Build(w.base)
	wantListed(t, "prd_list", toolData(t, s, "prd_list", kv{"workspace_id": ws.ID}), made.PrdID)
	if page := toolJSON[updatesPage](t, s, "project_updates", kv{"project_id": pr.ID, "before": *first.NextCursor}); len(page.Updates) != 1 {
		t.Fatalf("project_updates before the cursor = %+v, want the oldest note", page)
	}
	// rv5-PARB-04, PARA-06: the project and workspace reads and edits run on the agent token.
	if got := toolJSON[updatesPage](t, s, "project", kv{"project_id": pr.ID}); got.ID != pr.ID || got.Status != "paused" {
		t.Fatalf("project tool = %+v, want the paused project", got)
	}
	if got := toolJSON[updatesPage](t, s, "project_update", kv{"project_id": pr.ID, "status": "active"}); got.Status != "active" || got.Title != "Board "+sfx {
		t.Fatalf("project_update = %+v, want active with its title kept", got)
	}
	if got := toolJSON[updatesPage](t, s, "workspace_update", kv{"workspace_id": ws.ID, "description": "by the agent"}); got.Description != "by the agent" || got.Title != "Renamed "+sfx {
		t.Fatalf("workspace_update = %+v, want the new description and the title kept", got)
	}
	if got := toolJSON[updatesPage](t, s, "workspace", kv{"workspace_id": ws.ID}); got.Description != "by the agent" {
		t.Fatalf("workspace tool = %+v, want the agent's description", got)
	}
	note := toolJSON[doc](t, s, "update_add", kv{"project_id": pr.ID, "kind": "progress", "content": "agent note"})
	if got := toolJSON[doc](t, s, "update_edit", kv{"update_id": note.ID, "content": "agent note, corrected"}); got.Content != "agent note, corrected" {
		t.Fatalf("update_edit = %+v, want the corrected note", got)
	}

	// rv5-PARC-04: the human records what was observed, then confirms the node.
	leaf := cliJSON[doc](t, "roadmap", "add", "--title", "Leaf "+sfx, pr.ID)
	cliOut(t, "evidence", "add", "--detail", "Five users asked for it", "--observed", "2026-11-20", leaf.ID)
	if ev := cliOut(t, "evidence", leaf.ID); !strings.Contains(ev, "Five users asked for it") {
		t.Fatalf("evidence after evidence add = %s, want the observation", ev)
	}
	cliOut(t, "roadmap", "update", "--status", "in_progress", leaf.ID)
	cliOut(t, "roadmap", "update", "--status", "done", leaf.ID)
	if got := cliJSON[doc](t, "roadmap", "validate", "--reason", "Users kept it", leaf.ID); got.Status != "validated" {
		t.Fatalf("roadmap validate = %+v, want the node validated", got)
	}

	if out := strings.TrimSpace(cliOut(t, "project", "delete", pr.ID)); out != "deleted" {
		t.Fatalf("project delete printed %q, want deleted", out)
	}
	if res := runCLI(t, cliEnv(), "", "project", pr.ID); res.code == 0 {
		t.Fatalf("project after delete: exit 0\n%s\nwant the API's not-found error", res.output())
	}

	// rv5-PARA-03: the password comes through the prompt, and the login goes with the account.
	if res := runCLI(t, cliEnv(), w.password+"\n", "account", "delete"); res.code != 0 || strings.TrimSpace(res.stdout) != "account deleted" {
		t.Fatalf("account delete: exit %d\n%s\nwant account deleted", res.code, res.output())
	}
	if res := runCLI(t, cliEnv(), "", "whoami"); res.code == 0 {
		t.Fatalf("whoami after account delete: exit 0\n%s\nwant no session", res.output())
	}
}
