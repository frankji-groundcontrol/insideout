package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// earlyAnswer is what the user said in their very first turn. It shares almost
// no character bigrams with the filler turns after it, so only the real text can
// ground a quote of it.
const earlyAnswer = "our pottery studio keeps losing its kiln booking notes every monsoon"

// seedLongClarify persists a clarify conversation longer than the replay window:
// the first user turn is earlyAnswer, followed by exchanges of filler, all via
// the real store.
func seedLongClarify(t *testing.T, st *store.Store, ex *toolExecutor) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < historyLimit+5; i++ {
		text := fmt.Sprintf("reply %d", i)
		if i == 0 {
			text = earlyAnswer
		}
		if _, err := st.InsertAgentMessage(ctx, ex.actorID, ex.conversationID, string(RoleUser), text, nil, nil, nil); err != nil {
			t.Fatalf("insert user turn %d: %v", i, err)
		}
		if _, err := st.InsertAgentMessage(ctx, ex.actorID, ex.conversationID, string(RoleAssistant), fmt.Sprintf("noted %d", i), nil, nil, nil); err != nil {
			t.Fatalf("insert assistant turn %d: %v", i, err)
		}
	}
}

func recordEarlyAnswerArgs() string {
	b, _ := json.Marshal(map[string]string{"kind": "problem", "text": "booking notes get lost each monsoon", "quote": earlyAnswer})
	return string(b)
}

// plan.md: a fact's quote must come from "some prior user message in this
// conversation", with no window. A user turn older than the replay history must
// still ground it, or the draft gate's required facts can never be attested in
// a long clarify.
func TestRecordFact_GroundsAQuoteFromAnyEarlierUserTurn(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)
	seedLongClarify(t, st, ex) // ex.userMessages stays nil: the executor reads the conversation itself

	if _, err := ex.recordFact(ctx, recordEarlyAnswerArgs()); err != nil {
		t.Fatalf("record_fact with a verbatim quote of the user's first turn: %v", err)
	}
	lm, err := currentLedger(ctx, st, ex.actorID, ex.conversationID)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	if len(lm.Facts) != 1 || lm.Facts[0].Quote != earlyAnswer {
		t.Fatalf("ledger = %+v, want the one attested problem fact", lm.Facts)
	}
}

// The same through the real turn path: HandleMessage must not hand the executor
// a windowed copy of the history to ground against.
func TestHandleMessage_RecordFactGroundsAQuoteFromBeyondTheReplayWindow(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	userID, ex := seedConversation(t, st)
	seedLongClarify(t, st, ex)
	conv, err := st.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}

	calls := 0
	streamer := fakeStreamer{chat: func(string, func(string)) (Turn, error) {
		calls++
		if calls == 1 {
			return Turn{ToolCall: &ToolCallRequest{ID: "call-1", Name: "record_fact", Arguments: recordEarlyAnswerArgs()}}, nil
		}
		return Turn{Text: "got it"}, nil
	}}
	c := New(st, streamer, discardLog())
	c.HandleMessage(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil),
		ex.conversationID.String(), userID, conv.WorkspaceID.String(), "anything else you need?")

	lm, err := currentLedger(ctx, st, ex.actorID, ex.conversationID)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	if len(lm.Facts) != 1 || lm.Facts[0].Quote != earlyAnswer {
		msgs, _ := st.ListAgentMessages(ctx, ex.actorID, ex.conversationID)
		var toolResult string
		for _, m := range msgs {
			if m.Role == string(RoleTool) {
				toolResult = m.Content
			}
		}
		t.Fatalf("ledger = %+v after the model quoted the user's first turn, want the fact recorded; the model was told %q", lm.Facts, toolResult)
	}
}

// Each user turn is matched on its own. Joined into one haystack, the 70%
// bigram overlap would let a quote stitched from two different turns pass: a
// "verbatim" quote the user never said.
func TestRecordFact_DoesNotStitchAQuoteFromSeveralTurns(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)
	turnA, turnB := "aaaa bbbb cccc", "dddd eeee ffff"
	stitched := "aaaa bbbb dddd eeee"
	if quoteIsGrounded(stitched, turnA) || quoteIsGrounded(stitched, turnB) || !quoteIsGrounded(stitched, turnA+" "+turnB) {
		t.Fatal("fixture premise broken: the stitched quote must ground in the joined turns only, in neither alone")
	}
	for _, text := range []string{turnA, turnB} {
		if _, err := st.InsertAgentMessage(ctx, ex.actorID, ex.conversationID, string(RoleUser), text, nil, nil, nil); err != nil {
			t.Fatalf("insert user turn: %v", err)
		}
	}

	args, _ := json.Marshal(map[string]string{"kind": "problem", "text": "stitched", "quote": stitched})
	if _, err := ex.recordFact(ctx, string(args)); err == nil || !strings.Contains(err.Error(), "was not found") {
		t.Fatalf("record_fact with a quote stitched from two turns = %v, want it refused", err)
	}
}
