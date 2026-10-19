package store

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/github"
)

// Whole-change review of the repo binding (rv4-FR-F193, F205, F200, F121): the
// lookup that decides "who holds this repo" skipped every project without an
// owner, so a project that kept its repo_url but lost its owner (PATCH
// /projects/{pid} without an ownerId writes owner_id = NULL) stopped holding
// the repo, another project could take it, and deliveries moved without a word.
// A project holds its repo for as long as it carries the link; whom a delivery
// acts as is a separate question (its owner, while that owner is still a member
// of the workspace).

// rvGhBindOwnerless clears a project's owner the way PATCH /projects/{pid} does
// when no ownerId is sent: UpdateProject writes owner_id = NULL for a nil
// OwnerID. admin must own or administer the project.
func rvGhBindOwnerless(t *testing.T, st *Store, admin *User, p *Project) {
	t.Helper()
	f := ProjectUpdateFields{Title: p.Title, Description: p.Description, Status: p.Status}
	if _, err := st.UpdateProject(context.Background(), admin.ID, p.ID, f); err != nil {
		t.Fatalf("clear the owner of project %s: %v", p.ID, err)
	}
}

// rvGhBindActor is the user a delivery for this holder would act as. Printing
// the field keeps this file compiling whether OwnerID is a uuid.UUID or, once a
// holder may have nobody to act as, a *uuid.UUID that fmt prints as <nil>.
func rvGhBindActor(po ProjectOwner) string { return fmt.Sprint(po.OwnerID) }

func TestRvGhBindAnOwnerlessProjectStillHoldsItsRepo(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	a, b := mkTenant(t, st), mkTenant(t, st)
	owner, repo := uniqueSlug()
	url := github.CanonicalRepoURL(owner, repo)
	if _, err := st.SetProjectRepo(ctx, a.user.ID, a.project.ID, url); err != nil {
		t.Fatalf("A links the repo: %v", err)
	}
	rvGhBindOwnerless(t, st, a.user, a.project)

	if _, err := st.SetProjectRepo(ctx, b.user.ID, b.project.ID, url); !errors.Is(err, ErrRepoTaken) {
		t.Errorf("B linked the repo that ownerless project %s still carries (err %v), want ErrRepoTaken: two projects now hold one repo", a.project.ID, err)
	}

	// What deliveries go by names the same project, whatever B did.
	got, err := st.ProjectsByRepo(ctx, github.RepoURLFor(owner+"/"+repo))
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(got) != 1 || got[0].ProjectID != a.project.ID {
		t.Errorf("lookup named %d project(s) %+v, want exactly the ownerless holder %s", len(got), got, a.project.ID)
	}

	// The holder may link its own repo again (an admin repairing the project).
	if _, err := st.SetProjectRepo(ctx, a.user.ID, a.project.ID, url); err != nil {
		t.Errorf("the ownerless holder links its own repo again: %v", err)
	}
}

// Of rows that already share a repo, the one created first holds it, owned or
// not: the holder must not change when somebody's ownership does.
func TestRvGhBindTheEarliestProjectHoldsAMultiplyBoundRepoWhateverItsOwnerState(t *testing.T) {
	st := newTestStore(t)
	first, second := mkTenant(t, st), mkTenant(t, st) // created in this order
	owner, repo := uniqueSlug()
	url := github.CanonicalRepoURL(owner, repo)
	forceRepoURL(t, st, second, url)
	forceRepoURL(t, st, first, url)
	rvGhBindOwnerless(t, st, first.user, first.project)

	got, err := st.ProjectsByRepo(context.Background(), github.RepoURLFor(owner+"/"+repo))
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(got) != 1 || got[0].ProjectID != first.project.ID {
		t.Errorf("lookup named %d project(s) %+v, want only %s, the one created first (it has lost its owner, not its repo)", len(got), got, first.project.ID)
	}
}

// F121: RemoveMember leaves projects.owner_id alone. The project keeps its
// repo (an admin can read it and release it), but the member who left is no one
// a delivery may act as: every action as them is "not found" under RLS.
func TestRvGhBindAProjectWhoseOwnerLeftKeepsItsRepoWithNobodyToActAs(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	admin, member := mkUser(t, st), mkUser(t, st)
	stranger := mkTenant(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Leaver", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, member.ID, ws.Code); err != nil {
		t.Fatalf("member joins: %v", err)
	}
	p, err := st.CreateProject(ctx, member.ID, ws.ID, "Leaver project", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	owner, repo := uniqueSlug()
	url := github.CanonicalRepoURL(owner, repo)
	if _, err := st.SetProjectRepo(ctx, member.ID, p.ID, url); err != nil {
		t.Fatalf("member links the repo: %v", err)
	}

	lookup := func() []ProjectOwner {
		got, err := st.ProjectsByRepo(ctx, github.RepoURLFor(owner+"/"+repo))
		if err != nil {
			t.Fatalf("lookup: %v", err)
		}
		if len(got) != 1 || got[0].ProjectID != p.ID {
			t.Fatalf("lookup named %d project(s) %+v, want exactly %s", len(got), got, p.ID)
		}
		return got
	}
	if actor := rvGhBindActor(lookup()[0]); actor != member.ID.String() {
		t.Errorf("while the owner is a member the delivery acts as %s, want the owner %s", actor, member.ID)
	}

	if err := st.RemoveMember(ctx, admin.ID, ws.ID, member.ID); err != nil {
		t.Fatalf("remove the member: %v", err)
	}
	if actor := rvGhBindActor(lookup()[0]); actor != "<nil>" {
		t.Errorf("after the owner left, a delivery would act as %s: it can read nothing of the project, so the sync can only fail; want nobody to act as", actor)
	}

	// The repo stays held: the stranger cannot take it, and the admin can free it.
	if _, err := st.SetProjectRepo(ctx, stranger.user.ID, stranger.project.ID, url); !errors.Is(err, ErrRepoTaken) {
		t.Errorf("a stranger linked a repo the project of a departed owner holds: err %v, want ErrRepoTaken", err)
	}
	if _, err := st.SetProjectRepo(ctx, admin.ID, p.ID, ""); err != nil {
		t.Fatalf("admin releases the link: %v", err)
	}
	if _, err := st.SetProjectRepo(ctx, stranger.user.ID, stranger.project.ID, url); err != nil {
		t.Errorf("the freed repo is not linkable: %v", err)
	}
}

// rv4-FR-F199: a refused link says whether the caller may read the project that
// holds the repo. A holder in a workspace the caller is not in is
// ErrRepoHeldOutOfReach (still an ErrRepoTaken), so the API can answer as for a
// repo GitHub does not show; a holder the caller can read is a plain ErrRepoTaken.
func TestRvGhBindARefusedLinkSaysWhetherTheCallerMayReadTheHolder(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	admin, stranger := mkUser(t, st), mkTenant(t, st)
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Holder workspace", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	holder, err := st.CreateProject(ctx, admin.ID, ws.ID, "Holder project", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	owner, repo := uniqueSlug()
	url := github.CanonicalRepoURL(owner, repo)
	if _, err := st.SetProjectRepo(ctx, admin.ID, holder.ID, url); err != nil {
		t.Fatalf("holder links the repo: %v", err)
	}

	_, err = st.SetProjectRepo(ctx, stranger.user.ID, stranger.project.ID, url)
	if !errors.Is(err, ErrRepoHeldOutOfReach) || !errors.Is(err, ErrRepoTaken) {
		t.Errorf("a stranger's link: err = %v, want ErrRepoHeldOutOfReach, which is also an ErrRepoTaken", err)
	}

	// Once the stranger is a member of the holder's workspace the holder is
	// theirs to read, and the conflict is the specific one.
	if _, err := st.JoinWorkspace(ctx, stranger.user.ID, ws.Code); err != nil {
		t.Fatalf("stranger joins the holder's workspace: %v", err)
	}
	_, err = st.SetProjectRepo(ctx, stranger.user.ID, stranger.project.ID, url)
	if !errors.Is(err, ErrRepoTaken) || errors.Is(err, ErrRepoHeldOutOfReach) {
		t.Errorf("a member's link: err = %v, want a plain ErrRepoTaken: they can read the holder", err)
	}
}
