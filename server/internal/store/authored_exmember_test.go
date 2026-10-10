package store

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// SEC-11 followed through. 20261010120000 made the author branch of
// ideas_select, ideas_update and prds_insert need current membership, but four
// sibling policies still let authorship alone through: prds_update,
// project_updates_update, project_updates_delete and prd_revisions_insert. A
// member removed from a workspace kept the database-level right to edit the
// PRDs they wrote there, to edit or delete their project updates and to
// snapshot their PRDs. The store functions mask it through the runtime role
// (the SELECT policies hide the row first), so only a bare statement shows it.

type authoredFixture struct {
	ws          *Workspace
	admin, gone *User
	proj        *Project       // the admin's, so gone owns nothing but what they wrote on it
	prd         *Prd           // authored by gone, with one snapshot of their own (revision 2)
	edit, del   *ProjectUpdate // gone's notes on proj: one for the edit attempts, one for the delete
}

// newAuthoredFixture builds a workspace where gone, a plain member, authored a
// PRD, a snapshot of it and two project updates. gone is still a member on return.
func newAuthoredFixture(t *testing.T, st *Store) *authoredFixture {
	t.Helper()
	ctx := context.Background()
	f := &authoredFixture{admin: mkUser(t, st), gone: mkUser(t, st)}
	must := func(what string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}
	var err error
	f.ws, err = st.CreateWorkspace(ctx, f.admin.ID, "Authored ex-member WS", "")
	must("create workspace", err)
	_, err = st.JoinWorkspace(ctx, f.gone.ID, f.ws.Code)
	must("join workspace", err)
	f.proj, err = st.CreateProject(ctx, f.admin.ID, f.ws.ID, "Admin's project", "")
	must("create project", err)
	idea, err := st.CreateIdea(ctx, f.gone.ID, f.ws.ID, "Authored", "content")
	must("create idea", err)
	f.prd, _, err = st.ConvertIdea(ctx, f.gone.ID, idea.ID)
	must("convert idea", err)
	_, err = st.CreateRevision(ctx, f.gone.ID, f.prd.ID, nil)
	must("create revision", err)
	f.prd, err = st.GetPrdForMember(ctx, f.prd.ID, f.gone.ID)
	must("reload prd", err)
	f.edit, err = st.AddProjectUpdate(ctx, f.gone.ID, f.proj.ID, "note", "original")
	must("add update", err)
	f.del, err = st.AddProjectUpdate(ctx, f.gone.ID, f.proj.ID, "note", "doomed")
	must("add update", err)
	return f
}

// remove has the admin remove gone from the workspace.
func (f *authoredFixture) remove(t *testing.T, st *Store) {
	t.Helper()
	if err := st.RemoveMember(context.Background(), f.admin.ID, f.ws.ID, f.gone.ID); err != nil {
		t.Fatalf("remove member: %v", err)
	}
}

// wantUntouched reports unless the admin still sees everything exactly as
// gone left it: no edit, status change, snapshot, commit or delete took effect.
func (f *authoredFixture) wantUntouched(t *testing.T, st *Store) {
	t.Helper()
	ctx := context.Background()
	prd, err := st.GetPrdForMember(ctx, f.prd.ID, f.admin.ID)
	if err != nil {
		t.Fatalf("admin GetPrdForMember: %v", err)
	}
	if prd.Title != "Authored" || prd.Sections["background"] != "" || prd.Status != "draft" || prd.CurrentRevision != 2 {
		t.Errorf("a removed member's attempt changed the PRD: title %q, background %q, status %s, revision %d; want Authored, empty, draft, 2",
			prd.Title, prd.Sections["background"], prd.Status, prd.CurrentRevision)
	}
	revs, err := st.ListRevisions(ctx, f.admin.ID, f.prd.ID)
	if err != nil {
		t.Fatalf("admin ListRevisions: %v", err)
	}
	if len(revs) != 1 {
		t.Errorf("the PRD has %d snapshots, want the 1 gone took while a member", len(revs))
	}
	commits, err := st.ListPrdCommits(ctx, f.admin.ID, f.prd.ID)
	if err != nil {
		t.Fatalf("admin ListPrdCommits: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("a removed member committed the PRD: %d commits, want 0", len(commits))
	}
	ups, err := st.ListProjectUpdates(ctx, f.admin.ID, f.proj.ID, 0, nil)
	if err != nil {
		t.Fatalf("admin ListProjectUpdates: %v", err)
	}
	byID := map[uuid.UUID]ProjectUpdate{}
	for _, u := range ups {
		byID[u.ID] = u
	}
	if got, ok := byID[f.edit.ID]; !ok || got.Content != "original" {
		t.Errorf("a removed member's edit took effect on their update: present=%v, content %q, want original", ok, got.Content)
	}
	if _, ok := byID[f.del.ID]; !ok {
		t.Errorf("a removed member deleted their update")
	}
}

var errProbe = errors.New("probe: roll back")

// probe runs fn as user in a transaction that is always rolled back, so a
// statement that goes through leaves nothing behind for the next probe. It
// returns fn's own error, if any.
func probe(t *testing.T, st *Store, user uuid.UUID, fn func(ctx context.Context, tx pgx.Tx) error) error {
	t.Helper()
	ctx := context.Background()
	err := st.withUserContext(ctx, user, func(tx pgx.Tx) error {
		if err := fn(ctx, tx); err != nil {
			return err
		}
		return errProbe
	})
	if errors.Is(err, errProbe) {
		return nil
	}
	return err
}

// The database backstop, one policy at a time. Each statement is shaped so only
// the policy it names is consulted: a bare UPDATE or DELETE reads no column, so
// no SELECT policy is added to it, and an INSERT without RETURNING never
// consults prd_revisions_select.
func TestAuthoredExMember_PoliciesDeny(t *testing.T) {
	st := newTestStore(t)
	f := newAuthoredFixture(t, st)

	// gone is a fresh user, so these reach only the rows they wrote.
	touched := func(sql string) (n int64) {
		t.Helper()
		if err := probe(t, st, f.gone.ID, func(ctx context.Context, tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, sql)
			n = tag.RowsAffected()
			return err
		}); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		return n
	}
	const (
		editPrd = `UPDATE insideout.prds SET title = 'edited by a removed member'`              // prds_update
		editUpd = `UPDATE insideout.project_updates SET content = 'edited by a removed member'` // project_updates_update
		dropUpd = `DELETE FROM insideout.project_updates`                                       // project_updates_delete
	)
	snapshot := func() error { // prd_revisions_insert
		return probe(t, st, f.gone.ID, func(ctx context.Context, tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `
				INSERT INTO insideout.prd_revisions (prd_id, revision, sections, created_by)
				VALUES ($1, $2, '{}'::jsonb, $3)`, f.prd.ID, f.prd.CurrentRevision+1, f.gone.ID)
			return err
		})
	}

	// Controls: while a member all four statements work, so the zeros below can
	// only come from the policies.
	if n := touched(editPrd); n != 1 {
		t.Fatalf("member's bare UPDATE touched %d PRDs, want 1", n)
	}
	if n := touched(editUpd); n != 2 {
		t.Fatalf("member's bare UPDATE touched %d project updates, want 2", n)
	}
	if n := touched(dropUpd); n != 2 {
		t.Fatalf("member's bare DELETE removed %d project updates, want 2", n)
	}
	if err := snapshot(); err != nil {
		t.Fatalf("member's raw prd_revisions INSERT: %v", err)
	}

	f.remove(t, st)

	if n := touched(editPrd); n != 0 {
		t.Errorf("removed member's bare UPDATE still touched %d PRDs through prds_update, want 0", n)
	}
	if n := touched(editUpd); n != 0 {
		t.Errorf("removed member's bare UPDATE still touched %d project updates through project_updates_update, want 0", n)
	}
	if n := touched(dropUpd); n != 0 {
		t.Errorf("removed member's bare DELETE still removed %d project updates through project_updates_delete, want 0", n)
	}
	var pgErr *pgconn.PgError
	if err := snapshot(); !errors.As(err, &pgErr) || pgErr.Code != "42501" || !strings.Contains(pgErr.Message, "row-level security") {
		t.Errorf("removed member's raw prd_revisions INSERT: want an RLS violation (42501) from prd_revisions_insert, got %v", err)
	}
}

// Through the store as the runtime role a removed member is a stranger: the
// SELECT policies hide the row, so every call fails before reaching the write
// and nothing they attempted took effect. Membership, not the act of removal,
// is the gate: rejoining brings the rights back.
func TestAuthoredExMember_StoreRefuses(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	f := newAuthoredFixture(t, st)
	f.remove(t, st)

	wantNotFound := func(call string, err error) {
		t.Helper()
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("removed member %s: want ErrNotFound, got %v", call, err)
		}
	}
	_, err := st.UpdateSections(ctx, f.gone.ID, f.prd.ID, nil, map[string]string{"background": "hijacked"}, nil)
	wantNotFound("UpdateSections", err)
	_, err = st.UpdatePrdStatus(ctx, f.gone.ID, f.prd.ID, "reviewing")
	wantNotFound("UpdatePrdStatus", err)
	_, err = st.CreateRevision(ctx, f.gone.ID, f.prd.ID, nil)
	wantNotFound("CreateRevision", err)
	_, err = st.UpdateProjectUpdate(ctx, f.gone.ID, f.edit.ID, "hijacked")
	wantNotFound("UpdateProjectUpdate", err)
	wantNotFound("DeleteProjectUpdate", st.DeleteProjectUpdate(ctx, f.gone.ID, f.del.ID))
	// CreatePrdCommit reads the PRD FOR UPDATE and passes the empty result up as
	// pgx.ErrNoRows; the API never reaches it, GetPrdForMember answers 404 first.
	if _, err = st.CreatePrdCommit(ctx, f.gone.ID, f.prd.ID, "hijacked", "decision", "", nil, ""); err == nil {
		t.Errorf("removed member CreatePrdCommit succeeded, want it to fail")
	}

	f.wantUntouched(t, st)

	if _, err := st.JoinWorkspace(ctx, f.gone.ID, f.ws.Code); err != nil {
		t.Fatalf("rejoin: %v", err)
	}
	if _, err := st.UpdateSections(ctx, f.gone.ID, f.prd.ID, nil, map[string]string{"background": "back"}, nil); err != nil {
		t.Errorf("rejoined member UpdateSections: %v", err)
	}
	if _, err := st.UpdateProjectUpdate(ctx, f.gone.ID, f.edit.ID, "back"); err != nil {
		t.Errorf("rejoined member UpdateProjectUpdate: %v", err)
	}
	if err := st.DeleteProjectUpdate(ctx, f.gone.ID, f.del.ID); err != nil {
		t.Errorf("rejoined member DeleteProjectUpdate: %v", err)
	}
	if _, err := st.CreateRevision(ctx, f.gone.ID, f.prd.ID, nil); err != nil {
		t.Errorf("rejoined member CreateRevision: %v", err)
	}
	if _, err := st.UpdatePrdStatus(ctx, f.gone.ID, f.prd.ID, "reviewing"); err != nil {
		t.Errorf("rejoined member UpdatePrdStatus: %v", err)
	}
	if _, err := st.CreatePrdCommit(ctx, f.gone.ID, f.prd.ID, "v1", "decision", "", nil, ""); err != nil {
		t.Errorf("rejoined member CreatePrdCommit: %v", err)
	}
}
