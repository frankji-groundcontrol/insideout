package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ListMembers is member-only. A member sees the roster (join order); a
// non-member — a stranger, or someone who has since left — gets ErrForbidden,
// never an empty list that reads as "this workspace has no members".
func TestListMembers_MemberOnly(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	member := mkUser(t, st)
	stranger := mkUser(t, st)

	ws, err := st.CreateWorkspace(ctx, admin.ID, "Roster WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, member.ID, ws.Code); err != nil {
		t.Fatalf("join workspace: %v", err)
	}

	for _, viewer := range []*User{admin, member} {
		list, err := st.ListMembers(ctx, viewer.ID, ws.ID)
		if err != nil {
			t.Fatalf("member ListMembers: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("roster has %d rows, want 2", len(list))
		}
		if list[0].UserID != admin.ID || list[0].Role != "admin" {
			t.Fatalf("roster[0] = %s/%s, want the admin first (join order)", list[0].UserID, list[0].Role)
		}
		if list[1].UserID != member.ID || list[1].Role != "member" {
			t.Fatalf("roster[1] = %s/%s, want the member second", list[1].UserID, list[1].Role)
		}
	}

	list, err := st.ListMembers(ctx, stranger.ID, ws.ID)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("stranger ListMembers: want ErrForbidden, got err=%v with %d rows", err, len(list))
	}
	if list != nil {
		t.Fatalf("stranger ListMembers returned %d rows alongside the error, want none", len(list))
	}

	// Leaving revokes the roster too.
	if err := st.RemoveMember(ctx, member.ID, ws.ID, member.ID); err != nil {
		t.Fatalf("member leaves: %v", err)
	}
	if _, err := st.ListMembers(ctx, member.ID, ws.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("former member ListMembers: want ErrForbidden, got %v", err)
	}
}

// rv4-FR-F134, F141, F146: a workspace needs at least one admin. Role changes
// and approvals run on admins, no flow creates an admin after the workspace
// exists (JoinWorkspace writes members, the INSERT rule allows a first admin
// only), and an admin-less workspace can never be administered again. So the
// last admin can neither step down nor leave; the refusal is a conflict.

// rvStoreDbAdmins counts the admins of a workspace as viewer, who has to be a
// member, sees the roster.
func rvStoreDbAdmins(t *testing.T, st *Store, viewer, workspace uuid.UUID) int {
	t.Helper()
	list, err := st.ListMembers(context.Background(), viewer, workspace)
	if err != nil {
		t.Fatalf("roster: %v", err)
	}
	n := 0
	for _, m := range list {
		if m.Role == "admin" {
			n++
		}
	}
	return n
}

// rvStoreDbWantLastAdmin asserts the refusal: a conflict that says why.
func rvStoreDbWantLastAdmin(t *testing.T, what string, err error) {
	t.Helper()
	if !errors.Is(err, ErrConflict) || err == nil || !strings.Contains(err.Error(), "at least one admin") {
		t.Errorf("%s: want a conflict saying a workspace needs at least one admin, got %v", what, err)
	}
}

func TestRvStoreDbLastAdmin_SelfDemoteIsRefused(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	a, b := mkUser(t, st), mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, a.ID, "RV4 last admin, demote", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, b.ID, ws.Code); err != nil {
		t.Fatalf("join: %v", err)
	}

	rvStoreDbWantLastAdmin(t, "the only admin demoting themselves", st.UpdateMemberRole(ctx, a.ID, ws.ID, a.ID, "member"))
	if n := rvStoreDbAdmins(t, st, b.ID, ws.ID); n != 1 {
		t.Fatalf("%d admins after the refused demotion, want 1", n)
	}

	// None of these leaves the workspace without an admin.
	if err := st.UpdateMemberRole(ctx, a.ID, ws.ID, a.ID, "admin"); err != nil {
		t.Errorf("writing admin over admin: %v", err)
	}
	if err := st.UpdateMemberRole(ctx, a.ID, ws.ID, b.ID, "admin"); err != nil {
		t.Fatalf("promote b: %v", err)
	}
	if err := st.UpdateMemberRole(ctx, a.ID, ws.ID, a.ID, "member"); err != nil {
		t.Fatalf("a steps down while b is an admin: %v", err)
	}

	// b is now the only admin, and the rule holds for them. A plain member who
	// tries is still told they are not an admin, not that the workspace is short.
	rvStoreDbWantLastAdmin(t, "the new only admin demoting themselves", st.UpdateMemberRole(ctx, b.ID, ws.ID, b.ID, "member"))
	if err := st.UpdateMemberRole(ctx, a.ID, ws.ID, b.ID, "member"); !errors.Is(err, ErrForbidden) {
		t.Errorf("a plain member demoting the admin: want ErrForbidden, got %v", err)
	}
	if n := rvStoreDbAdmins(t, st, a.ID, ws.ID); n != 1 {
		t.Fatalf("%d admins at the end, want 1", n)
	}
}

func TestRvStoreDbLastAdmin_LeavingIsRefused(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	a, b, c := mkUser(t, st), mkUser(t, st), mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, a.ID, "RV4 last admin, leave", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	solo, err := st.CreateWorkspace(ctx, a.ID, "RV4 last admin, alone", "")
	if err != nil {
		t.Fatalf("create solo workspace: %v", err)
	}
	for _, u := range []*User{b, c} {
		if _, err := st.JoinWorkspace(ctx, u.ID, ws.Code); err != nil {
			t.Fatalf("join: %v", err)
		}
	}

	rvStoreDbWantLastAdmin(t, "the only admin leaving", st.RemoveMember(ctx, a.ID, ws.ID, a.ID))
	rvStoreDbWantLastAdmin(t, "the only member, an admin, leaving", st.RemoveMember(ctx, a.ID, solo.ID, a.ID))
	if n := rvStoreDbAdmins(t, st, a.ID, ws.ID); n != 1 {
		t.Fatalf("%d admins after the refused leave, want 1", n)
	}

	// A plain member leaves freely; with two admins one removes the other, and
	// then the survivor is the only admin again.
	if err := st.RemoveMember(ctx, c.ID, ws.ID, c.ID); err != nil {
		t.Errorf("a plain member leaving: %v", err)
	}
	if err := st.UpdateMemberRole(ctx, a.ID, ws.ID, b.ID, "admin"); err != nil {
		t.Fatalf("promote b: %v", err)
	}
	if err := st.RemoveMember(ctx, a.ID, ws.ID, b.ID); err != nil {
		t.Errorf("an admin removing the other admin: %v", err)
	}
	rvStoreDbWantLastAdmin(t, "the surviving admin leaving", st.RemoveMember(ctx, a.ID, ws.ID, a.ID))

	// The way out for a workspace nobody wants is deleting it.
	for _, id := range []uuid.UUID{ws.ID, solo.ID} {
		if err := st.DeleteWorkspace(ctx, a.ID, id); err != nil {
			t.Errorf("the creator deleting a workspace with a single admin: %v", err)
		}
	}
}

// Two admins demoting each other must not both succeed. The owner-side
// transaction holds b's row so that a's demotion of b is in flight (it has
// read two admins and waits at its UPDATE) while b's demotion of a starts: the
// interleaving that, unserialised, ends with no admin at all.
func TestRvStoreDbLastAdmin_TwoAdminsDemotingEachOtherKeepOne(t *testing.T) {
	st := newTestStore(t)
	owner := rvStoreDbOwnerStore(t)
	ctx := context.Background()
	a, b := mkUser(t, st), mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, a.ID, "RV4 last admin, race", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, b.ID, ws.Code); err != nil {
		t.Fatalf("join: %v", err)
	}
	if err := st.UpdateMemberRole(ctx, a.ID, ws.ID, b.ID, "admin"); err != nil {
		t.Fatalf("promote b: %v", err)
	}

	hold, err := owner.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin the holding transaction: %v", err)
	}
	defer hold.Rollback(ctx)
	var xid string
	if err := hold.QueryRow(ctx, `UPDATE insideout.workspace_memberships SET role = role WHERE workspace_id = $1 AND user_id = $2 RETURNING (txid_current() % 4294967296)::text`, ws.ID, b.ID).Scan(&xid); err != nil {
		t.Fatalf("hold b's row: %v", err)
	}

	type outcome struct {
		who string
		err error
	}
	done := make(chan outcome, 2)
	go func() { done <- outcome{"a demotes b", st.UpdateMemberRole(ctx, a.ID, ws.ID, b.ID, "member")} }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting int
		if err := owner.Pool.QueryRow(ctx, `SELECT count(*) FROM pg_locks WHERE locktype = 'transactionid' AND NOT granted AND transactionid::text = $1`, xid).Scan(&waiting); err != nil {
			t.Fatalf("look for the waiting update: %v", err)
		}
		if waiting > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("a's demotion of b never reached its UPDATE")
		}
		time.Sleep(10 * time.Millisecond)
	}
	go func() { done <- outcome{"b demotes a", st.UpdateMemberRole(ctx, b.ID, ws.ID, a.ID, "member")} }()
	time.Sleep(300 * time.Millisecond) // unserialised: b's call is finished by now; serialised: it is queued behind a's
	if err := hold.Rollback(ctx); err != nil {
		t.Fatalf("release b's row: %v", err)
	}

	var results []outcome
	for i := 0; i < 2; i++ {
		select {
		case o := <-done:
			results = append(results, o)
		case <-time.After(10 * time.Second):
			t.Fatalf("a call never returned; got %v", results)
		}
	}
	var admins int
	if err := owner.Pool.QueryRow(ctx, `SELECT count(*) FROM insideout.workspace_memberships WHERE workspace_id = $1 AND role = 'admin'`, ws.ID).Scan(&admins); err != nil {
		t.Fatalf("count admins: %v", err)
	}
	if admins < 1 {
		t.Errorf("both demotions went through, the workspace has %d admins (%+v)", admins, results)
	}
}
