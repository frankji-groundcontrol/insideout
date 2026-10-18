package store

import (
	"context"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/github"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// rv4-COMP-05: the webhook finds a project by matching the repo URL text, and
// two projects (of two owners, in two workspaces) could hold the same URL, so
// a signed push for one repo was delivered to both and each owner's timeline
// received the other's commit lines. The lookup now matches whatever letter
// case the owner typed, and names exactly one project even when old rows
// already share a repo; binding a second one is refused (see the api tests).

// tenant is an owner with a workspace and a project of their own.
type tenant struct {
	user    *User
	project *Project
}

func mkTenant(t *testing.T, st *Store) tenant {
	t.Helper()
	ctx := context.Background()
	u := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, u.ID, "Repo tenant", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	p, err := st.CreateProject(ctx, u.ID, ws.ID, "Repo tenant project", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return tenant{u, p}
}

// forceRepoURL writes projects.repo_url the way rows written before the PUT
// handler canonicalised it (or before a repo had a single holder) still sit in
// the table: as the owner, straight through RLS, past SetProjectRepo's checks.
func forceRepoURL(t *testing.T, st *Store, who tenant, url string) {
	t.Helper()
	ctx := context.Background()
	err := st.withUserContext(ctx, who.user.ID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE insideout.projects SET repo_url = $2 WHERE id = $1`, who.project.ID, url)
		if err == nil && tag.RowsAffected() != 1 {
			t.Fatalf("force repo_url: %d rows updated, want 1", tag.RowsAffected())
		}
		return err
	})
	if err != nil {
		t.Fatalf("force repo_url: %v", err)
	}
}

func uniqueSlug() (owner, repo string) {
	id := uuid.NewString()[:8]
	return "own" + id, "repo" + id
}

func TestProjectsByRepoIgnoresLetterCase(t *testing.T) {
	st := newTestStore(t)
	a := mkTenant(t, st)
	owner, repo := uniqueSlug()
	forceRepoURL(t, st, a, github.CanonicalRepoURL(owner, repo)) // typed in lower case

	// GitHub's full_name carries the repo's own capitalisation.
	got, err := st.ProjectsByRepo(context.Background(), github.RepoURLFor(strings.ToUpper(owner)+"/"+strings.ToUpper(repo)))
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(got) != 1 || got[0].ProjectID != a.project.ID {
		t.Errorf("lookup found %d project(s), want exactly project %s: a webhook for %s/%s would be dropped", len(got), a.project.ID, owner, repo)
	}
}

func TestProjectsByRepoNamesOneProjectWhenRowsShareARepo(t *testing.T) {
	st := newTestStore(t)
	first, second := mkTenant(t, st), mkTenant(t, st) // created in this order
	owner, repo := uniqueSlug()
	forceRepoURL(t, st, second, github.CanonicalRepoURL(owner, repo))
	forceRepoURL(t, st, first, github.CanonicalRepoURL(owner, repo))

	got, err := st.ProjectsByRepo(context.Background(), github.RepoURLFor(owner+"/"+repo))
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("lookup named %d projects, want exactly 1: every other owner would be sent this repo's events", len(got))
	}
	if got[0].ProjectID != first.project.ID {
		t.Errorf("lookup named project %s, want the one created first (%s): the choice must not depend on row order", got[0].ProjectID, first.project.ID)
	}
}

// An unlinked project has repo_url ”; the lookup helper must never read an
// empty URL as "every project without a repo".
func TestProjectsByRepoNeverMatchesTheEmptyURL(t *testing.T) {
	st := newTestStore(t)
	mkTenant(t, st) // a project with no repo linked
	got, err := st.ProjectsByRepo(context.Background(), "")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("lookup of the empty URL named %d project(s), want none", len(got))
	}
}
