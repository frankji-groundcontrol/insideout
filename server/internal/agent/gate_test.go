package agent

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// admitFixture is a fresh user (so the per-user rate limit starts at zero) and
// the workspace their runs belong to.
func admitFixture(t *testing.T, c *Coach) (userID, workspaceID uuid.UUID) {
	t.Helper()
	_, ex := seedConversation(t, c.store)
	conv, err := c.store.GetConversationForOwner(context.Background(), ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	return ex.actorID, conv.WorkspaceID
}

func admit(c *Coach, userID, workspaceID uuid.UUID) (*httptest.ResponseRecorder, func(error), bool) {
	rec := httptest.NewRecorder()
	done, ok := c.Admit(rec, httptest.NewRequest(http.MethodPost, "/", nil), workspaceID, userID, "roadmap:test")
	return rec, done, ok
}

// A planner call is a run like a coach turn: it counts toward the same 10/min
// limit, and the call after the limit is refused with the coach's own 429.
func TestAdmit_CountsTowardTheRateLimit(t *testing.T) {
	c := New(newTestStore(t), nil, discardLog())
	userID, workspaceID := admitFixture(t, c)

	for i := 0; i < 10; i++ {
		_, done, ok := admit(c, userID, workspaceID)
		if !ok {
			t.Fatalf("admit %d of 10 was refused", i+1)
		}
		done(nil)
	}
	rec, done, ok := admit(c, userID, workspaceID)
	if ok || done != nil {
		t.Fatal("the 11th admit within a minute was allowed")
	}
	if rec.Code != http.StatusTooManyRequests || !strings.Contains(rec.Body.String(), `"code":"APP_THROTTLE"`) {
		t.Fatalf("refusal = %d %s, want 429 APP_THROTTLE", rec.Code, rec.Body.String())
	}
}

// The permit is held until done, and a refused call leaves no run row behind.
func TestAdmit_HoldsAPermitUntilDone(t *testing.T) {
	t.Setenv("INSIDEOUT_AI_MAX_CONCURRENT", "1")
	t.Setenv("INSIDEOUT_AI_QUEUE_WAIT", "50ms")
	c := New(newTestStore(t), nil, discardLog())
	userID, workspaceID := admitFixture(t, c)

	_, first, ok := admit(c, userID, workspaceID)
	if !ok {
		t.Fatal("the first admit was refused")
	}
	rec, _, ok := admit(c, userID, workspaceID)
	if ok || rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), `"code":"CIRCUIT_OPEN"`) {
		t.Fatalf("second admit while the only permit is held = %d %s (ok=%v), want 503 CIRCUIT_OPEN", rec.Code, rec.Body.String(), ok)
	}
	first(nil)
	_, third, ok := admit(c, userID, workspaceID)
	if !ok {
		t.Fatal("admit after done released the permit was refused")
	}
	third(nil)

	var runs int
	if err := c.store.Pool.QueryRow(context.Background(), `SELECT count(*) FROM insideout.ai_runs WHERE user_id = $1`, userID).Scan(&runs); err != nil {
		t.Fatalf("count runs: %v", err)
	}
	if runs != 2 {
		t.Fatalf("%d run rows, want 2: the refused admit must not leave one behind", runs)
	}
}

// done finalizes the run the limiter counts, so no row is stranded "running".
// Only successes and non-provider failures are used here: this test must not
// write failures to the shared circuit-breaker row.
func TestAdmit_DoneRecordsTheOutcomeOnTheRun(t *testing.T) {
	c := New(newTestStore(t), nil, discardLog())
	userID, workspaceID := admitFixture(t, c)

	for _, callErr := range []error{nil, errors.New("planner returned no nodes"), context.Canceled} {
		_, done, ok := admit(c, userID, workspaceID)
		if !ok {
			t.Fatal("admit was refused")
		}
		done(callErr)
	}

	rows, err := c.store.Pool.Query(context.Background(), `SELECT status, coalesce(error_message, '') FROM insideout.ai_runs WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		t.Fatalf("read runs: %v", err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var status, msg string
		if err := rows.Scan(&status, &msg); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, status+":"+msg)
	}
	want := []string{"succeeded:", "failed:planner returned no nodes", "failed:context canceled"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("runs = %v, want %v", got, want)
	}
}

// A coach turn past the limit gets the same plain-JSON 429 before anything is
// written or sent to the provider; the gate it shares with Admit must stay in
// front of HandleMessage.
func TestHandleMessage_IsThrottledByThePerUserRateLimit(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	userID, ex := seedConversation(t, st)
	conv, err := st.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := st.CreateAIRun(ctx, conv.WorkspaceID, nil, ex.actorID, "earlier turn"); err != nil {
			t.Fatalf("create run %d: %v", i+1, err)
		}
	}
	streamer := fakeStreamer{chat: func(string, func(string)) (Turn, error) {
		t.Error("the provider was called for a throttled user")
		return Turn{}, nil
	}}
	c := New(st, streamer, discardLog())

	rec := httptest.NewRecorder()
	c.HandleMessage(rec, httptest.NewRequest(http.MethodPost, "/", nil),
		ex.conversationID.String(), userID, conv.WorkspaceID.String(), "one too many")

	if rec.Code != http.StatusTooManyRequests || !strings.Contains(rec.Body.String(), `"code":"APP_THROTTLE"`) ||
		!strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("throttled turn = %d %s %q, want a plain JSON 429 APP_THROTTLE", rec.Code, rec.Body.String(), rec.Header().Get("Content-Type"))
	}
	var msgs []store.AgentMessage
	if msgs, err = st.ListAgentMessages(ctx, ex.actorID, ex.conversationID); err != nil || len(msgs) != 0 {
		t.Fatalf("a throttled turn left %d message(s) behind (err %v), want none", len(msgs), err)
	}
}
