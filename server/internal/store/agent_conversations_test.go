package store

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// A coach conversation is owner-only (RLS), so the one ConvertIdea made for the
// converter is invisible to every other teammate: GET /prds/{id}/conversation
// 404s for them and the panel had no way to start one. EnsureConversationForPrd
// is that way — the PRD's author or a workspace admin, mirroring UpdateSections.
type coachPrd struct {
	ws        *Workspace
	converter *User
	prd       *Prd
	converted *AgentConversation // the converter's, made by ConvertIdea
}

func mkCoachPrd(t *testing.T, st *Store) coachPrd {
	t.Helper()
	ctx := context.Background()
	converter := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, converter.ID, "Coach Start WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, converter.ID, ws.ID, "Idea", "content")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	prd, conv, err := st.ConvertIdea(ctx, converter.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}
	return coachPrd{ws: ws, converter: converter, prd: prd, converted: conv}
}

// joinAs adds a fresh user to the workspace with the given role.
func (c coachPrd) joinAs(t *testing.T, st *Store, role string) *User {
	t.Helper()
	ctx := context.Background()
	u := mkUser(t, st)
	if _, err := st.JoinWorkspace(ctx, u.ID, c.ws.Code); err != nil {
		t.Fatalf("join workspace: %v", err)
	}
	if role != "member" {
		if err := st.UpdateMemberRole(ctx, c.converter.ID, c.ws.ID, u.ID, role); err != nil {
			t.Fatalf("set role %s: %v", role, err)
		}
	}
	return u
}

// countConversations counts userID's own rows for the PRD, under their RLS.
func countConversations(t *testing.T, st *Store, prdID, userID uuid.UUID) int {
	t.Helper()
	var n int
	err := st.withUserContext(context.Background(), userID, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(),
			`SELECT count(*) FROM insideout.agent_conversations WHERE prd_id = $1 AND user_id = $2`,
			prdID, userID,
		).Scan(&n)
	})
	if err != nil {
		t.Fatalf("count conversations: %v", err)
	}
	return n
}

func TestEnsureConversationForPrd_ConverterKeepsTheConvertedOne(t *testing.T) {
	st := newTestStore(t)
	c := mkCoachPrd(t, st)

	got, created, err := st.EnsureConversationForPrd(context.Background(), c.converter.ID, c.prd.ID)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if created || got.ID != c.converted.ID {
		t.Fatalf("converter got id=%s created=%v, want their ConvertIdea conversation %s with created=false", got.ID, created, c.converted.ID)
	}
	if n := countConversations(t, st, c.prd.ID, c.converter.ID); n != 1 {
		t.Fatalf("converter has %d conversations, want 1", n)
	}
}

func TestEnsureConversationForPrd_AnotherAdminGetsTheirOwn(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	c := mkCoachPrd(t, st)
	admin := c.joinAs(t, st, "admin")

	// Today the admin cannot even see the converter's conversation.
	if _, err := st.GetLatestConversationForPrd(ctx, c.prd.ID, admin.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("admin GetLatestConversationForPrd before ensure: want ErrNotFound, got %v", err)
	}

	first, created, err := st.EnsureConversationForPrd(ctx, admin.ID, c.prd.ID)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if !created {
		t.Fatal("first call: created = false, want true")
	}
	if first.ID == c.converted.ID || first.UserID != admin.ID || first.PrdID != c.prd.ID || first.WorkspaceID != c.ws.ID {
		t.Fatalf("admin's conversation = %+v, want a new one owned by the admin on this PRD", first)
	}

	again, created, err := st.EnsureConversationForPrd(ctx, admin.ID, c.prd.ID)
	if err != nil {
		t.Fatalf("ensure again: %v", err)
	}
	if created || again.ID != first.ID {
		t.Fatalf("repeat call got id=%s created=%v, want the same %s with created=false", again.ID, created, first.ID)
	}

	// And the existing read path now finds it.
	latest, err := st.GetLatestConversationForPrd(ctx, c.prd.ID, admin.ID)
	if err != nil || latest.ID != first.ID {
		t.Fatalf("GetLatestConversationForPrd after ensure = %v, %v; want %s", latest, err, first.ID)
	}
}

// The existence check is a plain SELECT, so without a lock concurrent callers
// all miss and each insert their own row (the table has no unique key on
// prd_id+user_id): a double tap or two tabs would leave the member with
// several conversations and the panel resuming an arbitrary one.
func TestEnsureConversationForPrd_ConcurrentCallsLeaveOneRow(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	c := mkCoachPrd(t, st)
	admin := c.joinAs(t, st, "admin")

	const callers = 8
	var wg sync.WaitGroup
	start := make(chan struct{}) // release all at once so the check→insert windows overlap
	ids := make([]uuid.UUID, callers)
	made := make([]bool, callers)
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			conv, created, err := st.EnsureConversationForPrd(ctx, admin.ID, c.prd.ID)
			errs[i], made[i] = err, created
			if conv != nil {
				ids[i] = conv.ID
			}
		}(i)
	}
	close(start)
	wg.Wait()

	creators := 0
	for i := 0; i < callers; i++ {
		if errs[i] != nil {
			t.Fatalf("caller %d: %v", i, errs[i])
		}
		if ids[i] != ids[0] {
			t.Fatalf("caller %d got conversation %s, caller 0 got %s: want one shared row", i, ids[i], ids[0])
		}
		if made[i] {
			creators++
		}
	}
	if creators != 1 {
		t.Fatalf("%d callers report created=true, want exactly 1", creators)
	}
	if n := countConversations(t, st, c.prd.ID, admin.ID); n != 1 {
		t.Fatalf("admin has %d conversations after %d concurrent starts, want 1", n, callers)
	}
}

func TestEnsureConversationForPrd_MemberAndStrangerAreRefused(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	c := mkCoachPrd(t, st)
	member := c.joinAs(t, st, "member")
	stranger := mkUser(t, st)

	// A plain member is in the workspace but is neither author nor admin.
	if _, _, err := st.EnsureConversationForPrd(ctx, member.ID, c.prd.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member: want ErrForbidden, got %v", err)
	}
	if n := countConversations(t, st, c.prd.ID, member.ID); n != 0 {
		t.Fatalf("a refused member has %d conversations, want 0", n)
	}
	// A non-member cannot see the PRD at all: existence is not disclosed.
	if _, _, err := st.EnsureConversationForPrd(ctx, stranger.ID, c.prd.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stranger: want ErrNotFound, got %v", err)
	}
	if _, _, err := st.EnsureConversationForPrd(ctx, c.converter.ID, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown PRD: want ErrNotFound, got %v", err)
	}
}
