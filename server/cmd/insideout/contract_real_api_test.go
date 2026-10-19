package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/api"
	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
)

// Review-4 TEST-04: the CLI, the apiclient and the MCP tools are otherwise
// tested only against handlers a test wrote by hand, so a route or a field
// renamed in internal/api breaks them with every Go suite green. This test
// points the real clients at the real thing: the API's own handler on a
// loopback listener over real PostgreSQL, with no route re-implemented. One user
// is driven through the apiclient, the CLI (this binary, re-executed as main)
// and the MCP tools, each reading what another wrote. DATABASE_URL-gated.
// (contract_real_api_mcp_test.go holds the MCP half.)

// doc is any one JSON object of the API, read for the few fields a test
// asserts on. A field the API renames reads as its zero value, which no
// assertion accepts.
type doc struct {
	ID, Email, Title, Content, Status, Kind, Mode, Decision string
	Code, MyRole, ProjectTitle                              string
	WorkspaceID, ProjectID, PrdID, ParentID, ConversationID string
	MemberCount, NodeCount, Revision                        int
	Recorded, Proposed                                      bool
	AppliedNodes                                            []string
	RevokedAt                                               *string
	Sections                                                map[string]string
	Invitation                                              *doc
}

// ids are the rows the CLI phase makes and the MCP phase then works on.
type ids struct{ workspace, project, prd, node, sfx string }

// contractWorld is one real API and one user who can sign in to it.
type contractWorld struct {
	st              *store.Store
	base            string // the /api/v1 base every surface is pointed at
	email, password string
}

func newContractWorld(t *testing.T) *contractWorld {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	testlock.RequireLocalHost(t, st.Pool)

	w := &contractWorld{st: st, email: "contract-" + uuid.NewString() + "@example.org", password: "pw-for-tests-" + uuid.NewString()}
	hash, err := auth.HashPassword(w.password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := st.CreateUser(ctx, w.email, hash, "contract-user"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	cfg := &config.Config{AccessTTL: time.Hour, RefreshTTL: time.Hour, PublicURL: "https://app.example.org"}
	srv := api.NewServer(st, auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour), cfg,
		slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil) // no coach, no planner: the LLM seams are not under test
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	w.base = ts.URL + "/api/v1"
	return w
}

// cliOut runs one verb of the real CLI against the saved login and returns its
// stdout; a non-zero exit fails the test.
func cliOut(t *testing.T, args ...string) string {
	t.Helper()
	res := runCLI(t, cliEnv(), "", args...)
	if res.code != 0 {
		t.Fatalf("insideout %s: exit %d\n%s", strings.Join(args, " "), res.code, res.output())
	}
	return res.stdout
}

func cliJSON[T any](t *testing.T, args ...string) T {
	t.Helper()
	return into[T](t, cliOut(t, args...))
}

func into[T any](t *testing.T, text string) T {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		t.Fatalf("not the JSON the test expects (%v): %s", err, text)
	}
	return v
}

// wantListed fails the test unless the JSON array in listing has an element
// with this id.
func wantListed(t *testing.T, what, listing, id string) {
	t.Helper()
	for _, e := range into[[]doc](t, listing) {
		if e.ID == id {
			return
		}
	}
	t.Fatalf("%s does not list %s: %s", what, id, listing)
}

func TestCLIAndMCPAgainstTheRealAPI(t *testing.T) {
	w := newContractWorld(t)

	// A real sign-in, saved the way `insideout login` saves it, is the CLI's session.
	c := apiclient.New(w.base)
	if err := c.Login(w.email, w.password); err != nil {
		t.Fatalf("login: %v", err)
	}
	claims, isJWT := apiclient.ParseTokenClaims(c.Token())
	if !isJWT || claims.Subject == "" || claims.Scope != "" || c.RefreshToken() == "" {
		t.Fatalf("login gave claims %+v (JWT %v), want a full-scope JWT and a refresh token", claims, isJWT)
	}
	writeClientConfig(t, clientconfig.Config{APIBase: w.base, Token: c.Token(), RefreshToken: c.RefreshToken()})

	var rows ids
	if !t.Run("CLI verbs", func(t *testing.T) { rows = w.cliVerbs(t, claims.Subject) }) {
		t.FailNow() // the later phases work on the rows this one made
	}
	t.Run("MCP tools", func(t *testing.T) { w.mcpTools(t, rows) })
	t.Run("MCP process that signs itself in", w.mcpSignedOut)
	t.Run("CLI admin verbs", w.cliAdmin)
}

// cliVerbs drives the CLI as an operator would: every verb is a child process
// that reads the saved login and prints what the API answered.
func (w *contractWorld) cliVerbs(t *testing.T, userID string) ids {
	sfx := uuid.NewString()[:8]
	if me := cliJSON[doc](t, "whoami"); me.ID != userID || me.Email != w.email {
		t.Fatalf("whoami = %+v, want the signed-in user %s", me, w.email)
	}

	ws := cliJSON[doc](t, "workspaces", "create", "--title", "Contract "+sfx, "--desc", "made by the contract test")
	if ws.ID == "" || ws.Title != "Contract "+sfx || ws.Code == "" || ws.MyRole != "admin" || ws.MemberCount != 1 {
		t.Fatalf("workspaces create = %+v, want its creator as the one admin, with a join code", ws)
	}
	wantListed(t, "workspaces", cliOut(t, "workspaces"), ws.ID)

	pr := cliJSON[doc](t, "projects", "create", "--title", "Board "+sfx, "--desc", "d", ws.ID)
	if pr.ID == "" || pr.WorkspaceID != ws.ID || pr.Title != "Board "+sfx {
		t.Fatalf("projects create = %+v, want a project of workspace %s", pr, ws.ID)
	}
	wantListed(t, "projects", cliOut(t, "projects", ws.ID), pr.ID)

	idea := cliJSON[doc](t, "idea", "create", "--title", "Idea "+sfx, "--content", "an idea from the CLI", ws.ID)
	if idea.ID == "" || idea.WorkspaceID != ws.ID || idea.Content != "an idea from the CLI" {
		t.Fatalf("idea create = %+v, want the idea in workspace %s", idea, ws.ID)
	}
	wantListed(t, "ideas", cliOut(t, "ideas", ws.ID), idea.ID)
	made := cliJSON[doc](t, "idea", "convert", idea.ID)
	if made.PrdID == "" || made.ConversationID == "" {
		t.Fatalf("idea convert = %+v, want a PRD and its coach conversation", made)
	}
	if prd := cliJSON[doc](t, "prd", made.PrdID); prd.ID != made.PrdID || prd.Title != idea.Title || prd.Status != "draft" {
		t.Fatalf("prd = %+v, want a draft titled as the idea", prd)
	}
	if edited := cliJSON[doc](t, "prd", "update", "--section", "background=Why now", made.PrdID); edited.Sections["background"] != "Why now" {
		t.Fatalf("prd update = %+v, want the background written", edited)
	}
	snap := cliJSON[doc](t, "snapshot", "--note", "first cut", made.PrdID)
	if revs := cliJSON[[]doc](t, "revisions", made.PrdID); snap.Revision == 0 || len(revs) != 1 || revs[0].Revision != snap.Revision {
		t.Fatalf("snapshot %+v, revisions %+v, want the one snapshot listed", snap, revs)
	}

	node := cliJSON[doc](t, "roadmap", "add", "--title", "Step "+sfx, pr.ID)
	if node.ID == "" || node.ProjectID != pr.ID || node.Title != "Step "+sfx {
		t.Fatalf("roadmap add = %+v, want a node of project %s", node, pr.ID)
	}
	wantListed(t, "roadmap list", cliOut(t, "roadmap", "list", pr.ID), node.ID)
	return ids{ws.ID, pr.ID, made.PrdID, node.ID, sfx}
}

// cliAdmin runs the admin verbs as a system admin, which only the owner role can make.
func (w *contractWorld) cliAdmin(t *testing.T) {
	ownerURL := os.Getenv("DATABASE_OWNER_URL")
	if ownerURL == "" {
		t.Skip("DATABASE_OWNER_URL not set, skipping: promoting an admin needs the owner role")
	}
	ctx := context.Background()
	if err := w.st.AttachMigratePool(ctx, ownerURL); err != nil {
		t.Fatalf("attach owner pool: %v", err)
	}
	if err := w.st.SetSystemRole(ctx, w.email, "admin"); err != nil { // read from the database on every request
		t.Fatalf("promote: %v", err)
	}

	invitee := "invitee-" + uuid.NewString() + "@example.org"
	made := cliJSON[doc](t, "invite", "--ttl", "24", invitee)
	if made.Invitation == nil || made.Invitation.Email != invitee || made.Code == "" {
		t.Fatalf("invite = %+v, want an invitation bound to %s and its one-time code", made, invitee)
	}
	id := made.Invitation.ID
	listed := func() doc {
		t.Helper()
		for _, i := range cliJSON[[]doc](t, "invites") {
			if i.ID == id {
				return i
			}
		}
		t.Fatalf("invites does not list %s", id)
		return doc{}
	}
	if listed().RevokedAt != nil {
		t.Fatalf("a new invitation is already revoked: %+v", listed())
	}
	if out := strings.TrimSpace(cliOut(t, "invite", "revoke", id)); out != "revoked" {
		t.Fatalf("invite revoke printed %q, want revoked", out)
	}
	if listed().RevokedAt == nil {
		t.Fatalf("invitation %s is not revoked after invite revoke", id)
	}
}
