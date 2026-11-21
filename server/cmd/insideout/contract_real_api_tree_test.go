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

	// The delegate, on the same machine, sees the branch as their tree.
	writeClientConfig(t, clientconfig.Config{APIBase: w.base, Token: delegate.Token(), RefreshToken: delegate.RefreshToken()})
	mine := cliJSON[[]struct {
		Node      treeNode
		ProjectID string
	}](t, "roadmap", "mine", ws.ID)
	if len(mine) != 1 || mine[0].Node.ID != branch.ID || mine[0].ProjectID != pr.ID || !mine[0].Node.CanWrite || mine[0].Node.Rollup.Leaves != 1 {
		t.Fatalf("roadmap mine = %+v, want the delegated branch, writable, over one leaf", mine)
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
