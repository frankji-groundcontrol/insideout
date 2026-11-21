package main

import (
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

	t.Cleanup(apiclient.ForgetAgentToken)
	s := mcpserver.Build(w.base)
	wantListed(t, "prd_list", toolData(t, s, "prd_list", kv{"workspace_id": ws.ID}), made.PrdID)
	if page := toolJSON[updatesPage](t, s, "project_updates", kv{"project_id": pr.ID, "before": *first.NextCursor}); len(page.Updates) != 1 {
		t.Fatalf("project_updates before the cursor = %+v, want the oldest note", page)
	}

	if out := strings.TrimSpace(cliOut(t, "project", "delete", pr.ID)); out != "deleted" {
		t.Fatalf("project delete printed %q, want deleted", out)
	}
	if res := runCLI(t, cliEnv(), "", "project", pr.ID); res.code == 0 {
		t.Fatalf("project after delete: exit 0\n%s\nwant the API's not-found error", res.output())
	}
}
