package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// rv4-SEC-04 at the store: a login starts a lineage ("session family"), every
// rotation carries it, and reuse of a token spent longer than the grace ago
// ends everything live in that lineage. token_hash is table-global unique and
// rotation revokes rather than deletes, so every hash carries a per-test suffix.

const reuseGrace = 10 * time.Second

// ageRevocations moves the revocation time of the user's revoked sessions into
// the past by d; the grace is judged against the database clock.
func ageRevocations(t *testing.T, st *Store, userID uuid.UUID, d time.Duration) {
	t.Helper()
	if _, err := st.Pool.Exec(context.Background(),
		`UPDATE insideout.sessions SET revoked_at = revoked_at - make_interval(secs => $2) WHERE user_id = $1 AND revoked_at IS NOT NULL`,
		userID, d.Seconds()); err != nil {
		t.Fatalf("age revocations: %v", err)
	}
}

func mustActive(t *testing.T, st *Store, hash, what string) *Session {
	t.Helper()
	sess, err := st.GetActiveSessionByHash(context.Background(), hash)
	if err != nil {
		t.Fatalf("%s should still be live: %v", what, err)
	}
	return sess
}

func mustBeDead(t *testing.T, st *Store, hash, what string) {
	t.Helper()
	if _, err := st.GetActiveSessionByHash(context.Background(), hash); !errors.Is(err, ErrNotFound) {
		t.Fatalf("%s should be revoked: want ErrNotFound, got %v", what, err)
	}
}

func TestSessionFamily_LoginStartsItAndRotationCarriesIt(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u := mkUser(t, st)
	exp := time.Now().Add(time.Hour)
	sfx := uuid.NewString()

	root, err := st.CreateSession(ctx, u.ID, "fam-root-"+sfx, exp)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if root.FamilyID != root.ID {
		t.Fatalf("a login's session is its own family root: family %s, id %s", root.FamilyID, root.ID)
	}
	next, err := st.RotateSession(ctx, root.ID, u.ID, "fam-next-"+sfx, exp)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	newest, err := st.RotateSession(ctx, next.ID, u.ID, "fam-newest-"+sfx, exp)
	if err != nil {
		t.Fatalf("rotate again: %v", err)
	}
	if next.FamilyID != root.ID || newest.FamilyID != root.ID {
		t.Fatalf("rotation must carry the family: root %s, next %s, newest %s", root.ID, next.FamilyID, newest.FamilyID)
	}
	if got := mustActive(t, st, "fam-newest-"+sfx, "the newest session"); got.FamilyID != root.ID {
		t.Fatalf("a stored session reads back family %s, want %s", got.FamilyID, root.ID)
	}

	other, err := st.CreateSession(ctx, u.ID, "fam-other-"+sfx, exp) // a second device signs in
	if err != nil {
		t.Fatalf("create second session: %v", err)
	}
	if other.FamilyID == root.FamilyID {
		t.Fatal("two logins share a family: reuse on one device would end the other")
	}
}

// Rows written before the column existed have no family_id. They read as a
// family of their own id, and their first rotation names the family.
func TestSessionFamily_PreUpgradeRowIsItsOwnFamily(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u := mkUser(t, st)
	sfx := uuid.NewString()

	var legacy uuid.UUID
	if err := st.Pool.QueryRow(ctx,
		`INSERT INTO insideout.sessions (user_id, token_hash, expires_at) VALUES ($1, $2, now() + interval '1 hour') RETURNING id`,
		u.ID, "fam-legacy-"+sfx).Scan(&legacy); err != nil {
		t.Fatalf("insert pre-upgrade session: %v", err)
	}
	if got := mustActive(t, st, "fam-legacy-"+sfx, "the pre-upgrade session"); got.FamilyID != legacy {
		t.Fatalf("pre-upgrade session reads family %s, want its own id %s", got.FamilyID, legacy)
	}
	next, err := st.RotateSession(ctx, legacy, u.ID, "fam-after-"+sfx, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("rotate pre-upgrade session: %v", err)
	}
	if next.FamilyID != legacy {
		t.Fatalf("successor family %s, want the pre-upgrade session's id %s", next.FamilyID, legacy)
	}

	// Replaying the pre-upgrade token later ends what descended from it.
	ageRevocations(t, st, u.ID, time.Hour)
	if _, ended, err := st.RevokeFamilyOnReuse(ctx, "fam-legacy-"+sfx, reuseGrace); err != nil || ended != 1 {
		t.Fatalf("replay of the pre-upgrade token: ended %d, err %v, want its 1 live successor", ended, err)
	}
	mustBeDead(t, st, "fam-after-"+sfx, "the successor of the replayed pre-upgrade session")
}

func TestRevokeFamilyOnReuse_EndsTheLineageAfterTheGrace(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u, bystander := mkUser(t, st), mkUser(t, st)
	exp := time.Now().Add(time.Hour)
	sfx := uuid.NewString()
	h := func(tag string) string { return "fam-" + tag + "-" + sfx }

	a0, err := st.CreateSession(ctx, u.ID, h("a0"), exp)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	a1, err := st.RotateSession(ctx, a0.ID, u.ID, h("a1"), exp)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if _, err := st.RotateSession(ctx, a1.ID, u.ID, h("a2"), exp); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if _, err := st.CreateSession(ctx, u.ID, h("b0"), exp); err != nil { // the user's other device
		t.Fatalf("create: %v", err)
	}
	if _, err := st.CreateSession(ctx, bystander.ID, h("c0"), exp); err != nil { // someone else entirely
		t.Fatalf("create: %v", err)
	}
	ageRevocations(t, st, u.ID, time.Hour)

	userID, ended, err := st.RevokeFamilyOnReuse(ctx, h("a0"), reuseGrace)
	if err != nil {
		t.Fatalf("revoke on reuse: %v", err)
	}
	if userID != u.ID || ended != 1 {
		t.Fatalf("ended %d sessions of %s, want 1 of %s (the newest of the lineage)", ended, userID, u.ID)
	}
	mustBeDead(t, st, h("a2"), "the newest session of the replayed lineage")
	mustActive(t, st, h("b0"), "the user's other device")
	mustActive(t, st, h("c0"), "another user's session")

	// A second replay finds a dead lineage: nothing to end, nothing to report.
	if _, ended, err := st.RevokeFamilyOnReuse(ctx, h("a1"), reuseGrace); err != nil || ended != 0 {
		t.Fatalf("replay into a dead lineage: ended %d, err %v, want 0 and no error", ended, err)
	}
}

// The grace is what keeps concurrent clients of one login alive, so its edge is
// pinned: a token spent 5 s ago is inside a 10 s grace, one spent 15 s ago is not.
func TestRevokeFamilyOnReuse_InsideTheGraceChangesNothing(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u := mkUser(t, st)
	exp := time.Now().Add(time.Hour)
	sfx := uuid.NewString()

	a0, err := st.CreateSession(ctx, u.ID, "fam-a0-"+sfx, exp)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := st.RotateSession(ctx, a0.ID, u.ID, "fam-a1-"+sfx, exp); err != nil {
		t.Fatalf("rotate: %v", err)
	}

	for _, age := range []time.Duration{0, 5 * time.Second} {
		ageRevocations(t, st, u.ID, age)
		if userID, ended, err := st.RevokeFamilyOnReuse(ctx, "fam-a0-"+sfx, reuseGrace); err != nil || ended != 0 || userID != uuid.Nil {
			t.Fatalf("spent %v ago, inside the %v grace: ended %d, err %v, want nothing", age, reuseGrace, ended, err)
		}
		mustActive(t, st, "fam-a1-"+sfx, "the winner's session")
	}

	ageRevocations(t, st, u.ID, 10*time.Second) // 15 s in all
	if _, ended, err := st.RevokeFamilyOnReuse(ctx, "fam-a0-"+sfx, reuseGrace); err != nil || ended != 1 {
		t.Fatalf("spent 15 s ago, outside the grace: ended %d, err %v, want 1", ended, err)
	}
	mustBeDead(t, st, "fam-a1-"+sfx, "the winner's session after the grace")
}

func TestRevokeFamilyOnReuse_IgnoresTokensThatAreNotSpent(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u := mkUser(t, st)
	sfx := uuid.NewString()

	live, err := st.CreateSession(ctx, u.ID, "fam-live-"+sfx, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := st.CreateSession(ctx, u.ID, "fam-stale-"+sfx, time.Now().Add(-time.Hour)); err != nil { // expired, never revoked
		t.Fatalf("create: %v", err)
	}
	loggedOut, err := st.CreateSession(ctx, u.ID, "fam-out-"+sfx, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := st.RevokeSession(ctx, loggedOut.ID); err != nil {
		t.Fatalf("logout: %v", err)
	}
	ageRevocations(t, st, u.ID, time.Hour)

	for name, hash := range map[string]string{
		"a live token":    "fam-live-" + sfx,
		"an unknown hash": "fam-unknown-" + sfx,
		"an expired one":  "fam-stale-" + sfx,
		"a logged-out one (its lineage has nothing else live)": "fam-out-" + sfx,
	} {
		if userID, ended, err := st.RevokeFamilyOnReuse(ctx, hash, reuseGrace); err != nil || ended != 0 || userID != uuid.Nil {
			t.Errorf("%s: ended %d, err %v, want nothing", name, ended, err)
		}
	}
	if got := mustActive(t, st, "fam-live-"+sfx, "the live session"); got.ID != live.ID {
		t.Fatalf("live session changed identity")
	}
}
