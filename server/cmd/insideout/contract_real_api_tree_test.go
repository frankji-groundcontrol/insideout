package main

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// rv5-PARC-01, TREEB-01, TREEB-04: the delegation walk on the real API, from
// the CLI only. The project owner finds a teammate with `members`, delegates a
// branch to them, and the teammate, signed in on the same machine, finds it
// with `roadmap mine`, works inside it and hands it back. DATABASE_URL-gated.

// treeNode is a roadmap node view, read for the fields the walk asserts on.
type treeNode struct {
	ID, ProjectID, Title, Acceptance string
	OwnerID                          *string
	CanWrite                         bool
	DependsOn                        []string
	Rollup                           struct{ Leaves, Done int }
}

func TestCLIDelegatesANodeOnTheRealAPI(t *testing.T) {
	w := newContractWorld(t)
	ctx := context.Background()
	owner := apiclient.New(w.base)
	if err := owner.Login(w.email, w.password); err != nil {
		t.Fatalf("login: %v", err)
	}
	claims, _ := apiclient.ParseTokenClaims(owner.Token())
	ownerID := claims.Subject
	writeClientConfig(t, clientconfig.Config{APIBase: w.base, Token: owner.Token(), RefreshToken: owner.RefreshToken()})

	sfx := uuid.NewString()[:8]
	ws := cliJSON[doc](t, "workspaces", "create", "--title", "Tree "+sfx)
	pr := cliJSON[doc](t, "projects", "create", "--title", "Board "+sfx, ws.ID)
	branch := cliJSON[doc](t, "roadmap", "add", "--title", "Branch "+sfx, pr.ID)
	leaf := cliJSON[doc](t, "roadmap", "add", "--title", "Leaf "+sfx, "--parent", branch.ID, pr.ID)
	other := cliJSON[doc](t, "roadmap", "add", "--title", "Other "+sfx, pr.ID)

	// The delegate signs up and joins with the workspace code.
	email, password := "delegate-"+uuid.NewString()+"@example.org", "pw-for-tests-"+uuid.NewString()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := w.st.CreateUser(ctx, email, hash, "delegate-user"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	delegate := apiclient.New(w.base)
	if err := delegate.Login(email, password); err != nil {
		t.Fatalf("delegate login: %v", err)
	}
	if _, err := delegate.JoinWorkspace(ws.Code); err != nil {
		t.Fatalf("join: %v", err)
	}

	var delegateID string
	for _, m := range cliJSON[[]struct{ UserID, Email string }](t, "members", ws.ID) {
		if m.Email == email {
			delegateID = m.UserID
		}
	}
	if delegateID == "" {
		t.Fatalf("members does not list the delegate %s", email)
	}

	handed := cliJSON[treeNode](t, "roadmap", "delegate", "--to", delegateID, "--note", "yours now", branch.ID)
	if handed.OwnerID == nil || *handed.OwnerID != delegateID || !handed.CanWrite {
		t.Fatalf("roadmap delegate = %+v, want the delegate as owner and the plan writer still writing", handed)
	}
	history := cliJSON[[]struct{ Kind, ToUserID, Note string }](t, "roadmap", "assignments", branch.ID)
	if len(history) != 1 || history[0].Kind != "delegate" || history[0].ToUserID != delegateID || history[0].Note != "yours now" {
		t.Fatalf("roadmap assignments = %+v, want the one delegation", history)
	}
	cliJSON[treeNode](t, "roadmap", "depend", "--on", other.ID, leaf.ID)

	// The delegate, on the same machine, sees the branch as their tree.
	writeClientConfig(t, clientconfig.Config{APIBase: w.base, Token: delegate.Token(), RefreshToken: delegate.RefreshToken()})
	mine := cliJSON[[]struct {
		Node      treeNode
		ProjectID string
	}](t, "roadmap", "mine", ws.ID)
	if len(mine) != 1 || mine[0].Node.ID != branch.ID || mine[0].ProjectID != pr.ID || !mine[0].Node.CanWrite || mine[0].Node.Rollup.Leaves != 1 {
		t.Fatalf("roadmap mine = %+v, want the delegated branch, writable, over one leaf", mine)
	}
	// With no workspace id, mine reads every workspace; the branch's first
	// glance names the unowned root its leaf waits on.
	all := cliJSON[[]struct {
		Node        treeNode
		WorkspaceID string
		WaitsOn     []struct{ ID, Title, Status string }
	}](t, "roadmap", "mine")
	if len(all) != 1 || all[0].Node.ID != branch.ID || all[0].WorkspaceID != ws.ID ||
		len(all[0].WaitsOn) != 1 || all[0].WaitsOn[0].ID != other.ID || all[0].WaitsOn[0].Status != "pending" {
		t.Fatalf("roadmap mine (all workspaces) = %+v, want the branch waiting on Other", all)
	}
	// The unowned leaf opens as the delegate sees it, through the branch;
	// the unowned root Other falls to the project owner.
	type glanceOut struct {
		Node     treeNode
		InCharge *struct {
			UserID       string
			Via          *struct{ ID, Title string }
			ProjectOwner bool
		}
		WaitsOn []struct{ ID string }
	}
	g := cliJSON[glanceOut](t, "roadmap", "glance", leaf.ID)
	if g.Node.ID != leaf.ID || g.InCharge == nil || g.InCharge.UserID != delegateID || g.InCharge.Via == nil ||
		g.InCharge.Via.ID != branch.ID || g.InCharge.ProjectOwner || len(g.WaitsOn) != 1 || g.WaitsOn[0].ID != other.ID {
		t.Fatalf("roadmap glance leaf = %+v, want the delegate via the branch, waiting on Other", g)
	}
	if g := cliJSON[glanceOut](t, "roadmap", "glance", other.ID); g.InCharge == nil || g.InCharge.UserID != ownerID ||
		!g.InCharge.ProjectOwner || g.InCharge.Via != nil {
		t.Fatalf("roadmap glance Other = %+v, want the project owner", g.InCharge)
	}
	edited := cliJSON[treeNode](t, "roadmap", "update", "--acceptance", "Ships on Friday", "--owner", "me", leaf.ID)
	if edited.Acceptance != "Ships on Friday" || edited.OwnerID == nil || *edited.OwnerID != delegateID {
		t.Fatalf("roadmap update = %+v, want the acceptance and the delegate as owner of the leaf", edited)
	}
	shown := cliJSON[struct {
		Node      treeNode
		Ancestors []struct{ ID string }
	}](t, "roadmap", "show", leaf.ID)
	if shown.Node.ID != leaf.ID || len(shown.Ancestors) != 1 || shown.Ancestors[0].ID != branch.ID {
		t.Fatalf("roadmap show = %+v, want the leaf under the branch", shown)
	}
	if sub := cliJSON[[]treeNode](t, "roadmap", "list", "--root", branch.ID, pr.ID); len(sub) != 2 {
		t.Fatalf("roadmap list --root = %+v, want the branch and its leaf", sub)
	}

	// Work in their branch waits on a node outside it.
	if waits := cliJSON[treeNode](t, "roadmap", "depend", "--on", other.ID, leaf.ID); len(waits.DependsOn) != 1 || waits.DependsOn[0] != other.ID {
		t.Fatalf("roadmap depend = %+v, want the leaf waiting on %s", waits, other.ID)
	}
	// The top level is the two roots, Branch after Other; Branch opens on its leaf.
	type stepsOut struct {
		Root  *treeNode
		Steps []struct {
			Node         treeNode
			Stage        int
			After        []string
			WaitsOutside []struct{ ID string }
		}
	}
	top := cliJSON[stepsOut](t, "roadmap", "steps", pr.ID)
	if top.Root != nil || len(top.Steps) != 2 || top.Steps[0].Node.ID != other.ID || top.Steps[1].Node.ID != branch.ID ||
		top.Steps[1].Stage != 1 || len(top.Steps[1].After) != 1 || top.Steps[1].After[0] != other.ID {
		t.Fatalf("roadmap steps = %+v, want Other then Branch after it", top)
	}
	in := cliJSON[stepsOut](t, "roadmap", "steps", "--root", branch.ID, pr.ID)
	if in.Root == nil || in.Root.ID != branch.ID || len(in.Steps) != 1 || in.Steps[0].Node.ID != leaf.ID ||
		len(in.Steps[0].WaitsOutside) != 1 || in.Steps[0].WaitsOutside[0].ID != other.ID {
		t.Fatalf("roadmap steps --root = %+v, want the leaf waiting outside on Other", in)
	}
	if out := cliOut(t, "roadmap", "undepend", "--on", other.ID, leaf.ID); strings.TrimSpace(out) != "removed" {
		t.Fatalf("roadmap undepend printed %q", out)
	}
	if n := cliJSON[struct{ Node treeNode }](t, "roadmap", "show", leaf.ID).Node; len(n.DependsOn) != 0 {
		t.Fatalf("after undepend the leaf still waits on %v", n.DependsOn)
	}

	back := cliJSON[treeNode](t, "roadmap", "handback", "--note", "over to you", branch.ID)
	if back.OwnerID == nil || *back.OwnerID != ownerID || back.CanWrite {
		t.Fatalf("roadmap handback = %+v, want the owner back and the delegate no longer writing", back)
	}
}
