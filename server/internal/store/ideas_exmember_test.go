package store

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// SEC-11: authorship decides which member may edit, drop or convert an idea,
// never whether the actor still belongs to the workspace. The author branches
// of the store functions and of ideas_select / ideas_update / prds_insert never
// asked, so a removed member kept read, edit, drop and convert rights over
// their old ideas — and convert minted a PRD in a workspace they had left.

type exMemberFixture struct {
	ws          *Workspace
	admin, gone *User
	edit, conv  *Idea // both authored by gone: one for update/drop, one for convert
}

// newExMemberFixture builds a workspace where gone, a plain member, authored
// two ideas. gone is still a member on return.
func newExMemberFixture(t *testing.T, st *Store) *exMemberFixture {
	t.Helper()
	ctx := context.Background()
	f := &exMemberFixture{admin: mkUser(t, st), gone: mkUser(t, st)}
	var err error
	if f.ws, err = st.CreateWorkspace(ctx, f.admin.ID, "Ex-member WS", ""); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err = st.JoinWorkspace(ctx, f.gone.ID, f.ws.Code); err != nil {
		t.Fatalf("join workspace: %v", err)
	}
	if f.edit, err = st.CreateIdea(ctx, f.gone.ID, f.ws.ID, "Keep", "original"); err != nil {
		t.Fatalf("create idea: %v", err)
	}
	if f.conv, err = st.CreateIdea(ctx, f.gone.ID, f.ws.ID, "Convert", "original"); err != nil {
		t.Fatalf("create idea: %v", err)
	}
	return f
}

// remove has the admin remove gone from the workspace.
func (f *exMemberFixture) remove(t *testing.T, st *Store) {
	t.Helper()
	if err := st.RemoveMember(context.Background(), f.admin.ID, f.ws.ID, f.gone.ID); err != nil {
		t.Fatalf("remove member: %v", err)
	}
}

// wantUntouched reports unless the admin still sees both ideas exactly as gone
// created them: no edit, no drop, no conversion took effect.
func (f *exMemberFixture) wantUntouched(t *testing.T, st *Store) {
	t.Helper()
	ctx := context.Background()
	edit, err := st.GetIdeaForMember(ctx, f.edit.ID, f.admin.ID)
	if err != nil {
		t.Fatalf("admin GetIdeaForMember: %v", err)
	}
	if edit.Title != "Keep" || edit.Content != "original" || edit.Status != "inbox" {
		t.Errorf("a removed member's attempt took effect: idea is now %q/%q/%s, want Keep/original/inbox", edit.Title, edit.Content, edit.Status)
	}
	conv, err := st.GetIdeaForMember(ctx, f.conv.ID, f.admin.ID)
	if err != nil {
		t.Fatalf("admin GetIdeaForMember: %v", err)
	}
	if conv.Status != "inbox" || conv.PrdID != nil {
		t.Errorf("a removed member converted an idea: status %s, prd %v, want inbox with no PRD", conv.Status, conv.PrdID)
	}
}

// The database backstop, one policy at a time. Each statement is shaped so only
// the policy it names is consulted: a bare UPDATE reads no column, so no SELECT
// policy is added to it, and an INSERT without RETURNING never consults
// prds_select.
func TestIdeasExMember_PoliciesDeny(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	f := newExMemberFixture(t, st)

	asGone := func(fn func(tx pgx.Tx) error) error { return st.withUserContext(ctx, f.gone.ID, fn) }
	visible := func() (n int) { // ideas_select
		if err := asGone(func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, `SELECT count(*) FROM insideout.ideas WHERE workspace_id = $1`, f.ws.ID).Scan(&n)
		}); err != nil {
			t.Fatalf("count ideas: %v", err)
		}
		return n
	}
	touched := func() (n int64) { // ideas_update (gone is a fresh user: only their two ideas are in reach)
		if err := asGone(func(tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, `UPDATE insideout.ideas SET content = 'edited by a removed member'`)
			n = tag.RowsAffected()
			return err
		}); err != nil {
			t.Fatalf("bare UPDATE: %v", err)
		}
		return n
	}
	plant := func() error { // prds_insert
		return asGone(func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `INSERT INTO insideout.prds (workspace_id, author_id, title) VALUES ($1, $2, 'planted')`, f.ws.ID, f.gone.ID)
			return err
		})
	}

	// Controls: while a member all three statements work, so the zeros below
	// can only come from the policies.
	if n := visible(); n != 2 {
		t.Fatalf("member sees %d ideas, want 2", n)
	}
	if n := touched(); n != 2 {
		t.Fatalf("member's bare UPDATE touched %d ideas, want 2", n)
	}
	if err := plant(); err != nil {
		t.Fatalf("member's raw prds INSERT: %v", err)
	}

	f.remove(t, st)

	if n := visible(); n != 0 {
		t.Errorf("removed member still sees %d ideas through ideas_select, want 0", n)
	}
	if n := touched(); n != 0 {
		t.Errorf("removed member's bare UPDATE still touched %d ideas through ideas_update, want 0", n)
	}
	var pgErr *pgconn.PgError
	if err := plant(); !errors.As(err, &pgErr) || pgErr.Code != "42501" || !strings.Contains(pgErr.Message, "row-level security") {
		t.Errorf("removed member's raw prds INSERT: want an RLS violation (42501) from prds_insert, got %v", err)
	}
}

// Through the store as the runtime role a removed member is a stranger: RLS
// hides the idea, so every call answers "not found" (existence is not
// disclosed) and nothing they attempted took effect. Membership, not the act
// of removal, is the gate: rejoining brings the idea back.
func TestIdeasExMember_StoreAnswersNotFound(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	f := newExMemberFixture(t, st)
	f.remove(t, st)

	wantNotFound := func(who string, err error) {
		t.Helper()
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: want ErrNotFound, got %v", who, err)
		}
	}
	_, err := st.GetIdeaForMember(ctx, f.edit.ID, f.gone.ID)
	wantNotFound("removed member GetIdeaForMember", err)
	_, err = st.UpdateIdea(ctx, f.gone.ID, f.edit.ID, "hijacked", "x")
	wantNotFound("removed member UpdateIdea", err)
	wantNotFound("removed member DropIdea", st.DropIdea(ctx, f.gone.ID, f.edit.ID))
	_, _, err = st.ConvertIdea(ctx, f.gone.ID, f.conv.ID)
	wantNotFound("removed member ConvertIdea", err)

	f.wantUntouched(t, st)

	if _, err := st.JoinWorkspace(ctx, f.gone.ID, f.ws.Code); err != nil {
		t.Fatalf("rejoin: %v", err)
	}
	if _, err := st.GetIdeaForMember(ctx, f.edit.ID, f.gone.ID); err != nil {
		t.Errorf("rejoined member GetIdeaForMember: %v", err)
	}
}

// The Go layer must refuse on its own, not lean on RLS. Run the same store code
// as insideout_owner (BYPASSRLS), where no policy can hide the row: only the
// explicit membership checks stand between a removed member and their ideas.
func TestIdeasExMember_GoLayerDenies(t *testing.T) {
	ownerURL := os.Getenv("DATABASE_OWNER_URL")
	if ownerURL == "" {
		t.Skip("DATABASE_OWNER_URL not set, skipping the RLS-bypassing check of the Go layer")
	}
	ctx := context.Background()
	app := newTestStore(t)
	owner, err := Open(ctx, ownerURL)
	if err != nil {
		t.Fatalf("open owner store: %v", err)
	}
	t.Cleanup(owner.Close)
	var bypass bool
	if err := owner.Pool.QueryRow(ctx, `SELECT rolbypassrls OR rolsuper FROM pg_roles WHERE rolname = current_user`).Scan(&bypass); err != nil || !bypass {
		t.Fatalf("DATABASE_OWNER_URL role must bypass RLS for this test (bypass=%v, err=%v)", bypass, err)
	}

	f := newExMemberFixture(t, app)
	f.remove(t, app)

	if idea, err := owner.GetIdeaForMember(ctx, f.edit.ID, f.gone.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("removed member GetIdeaForMember: want ErrNotFound, got err=%v (idea returned: %v)", err, idea != nil)
	}
	if _, err := owner.UpdateIdea(ctx, f.gone.ID, f.edit.ID, "hijacked", "x"); !errors.Is(err, ErrForbidden) {
		t.Errorf("removed member UpdateIdea: want ErrForbidden, got %v", err)
	}
	if err := owner.DropIdea(ctx, f.gone.ID, f.edit.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("removed member DropIdea: want ErrForbidden, got %v", err)
	}
	if prd, _, err := owner.ConvertIdea(ctx, f.gone.ID, f.conv.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("removed member ConvertIdea: want ErrForbidden, got err=%v (minted PRD: %v)", err, prd != nil)
	}

	f.wantUntouched(t, owner)
}
