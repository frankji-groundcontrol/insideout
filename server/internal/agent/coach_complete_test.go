package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// walkToFinalize advances a freshly seeded conversation through the real
// stage gates — the four attested fact kinds, every section written, a critic
// skip marker — confirming each proposal, so HandleMessage sees a
// finalize-stage conversation without any direct SQL.
func walkToFinalize(t *testing.T, st *store.Store, ex *toolExecutor) {
	t.Helper()
	ctx := context.Background()

	// clarify -> draft: the four fact kinds the gate demands, each quoted
	// from something the user actually said this turn.
	ex.userMessages = []string{
		"nobody can reuse the workshop notes afterwards",
		"our fifteen designers scroll a giant thread instead",
		"the old wiki died, so they cope by asking around",
		"we need it before the next cohort starts",
	}
	must := func(kind, text, quote string) {
		t.Helper()
		if _, err := ex.recordFact(ctx, `{"kind":"`+kind+`","text":"`+text+`","quote":"`+quote+`"}`); err != nil {
			t.Fatalf("record_fact %s: %v", kind, err)
		}
	}
	must("problem", "workshop notes are unusable afterwards", "nobody can reuse the workshop notes afterwards")
	must("segment", "a fifteen-designer cohort", "our fifteen designers scroll a giant thread instead")
	must("alternative", "asking around", "the old wiki died, so they cope by asking around")
	must("whynow", "before the next cohort starts", "we need it before the next cohort starts")

	confirm := func(next string) {
		t.Helper()
		if _, err := ex.advanceStage(ctx, `{"next":"`+next+`"}`); err != nil {
			t.Fatalf("advance to %s: %v", next, err)
		}
		if err := st.ConfirmConversationStage(ctx, ex.actorID, ex.conversationID, next, ApplyStageConfirmation); err != nil {
			t.Fatalf("confirm %s: %v", next, err)
		}
		ex.currentStage = next
	}
	confirm(StageDraft)

	// draft -> critique: every section non-empty.
	for _, section := range store.PrdSectionKeys {
		if _, err := ex.updateSection(ctx, `{"section":"`+section+`","markdown":"content for `+section+`"}`); err != nil {
			t.Fatalf("update section %s: %v", section, err)
		}
	}
	confirm(StageCritique)

	// critique -> finalize: the skip marker stands in for a critic pass.
	conv, err := st.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	lm, extra := loadLedger(conv.Meta)
	saveCriticState(extra, criticState{Skipped: "contention"})
	metaJSON, err := saveLedger(lm, extra)
	if err != nil {
		t.Fatalf("save ledger: %v", err)
	}
	if err := st.UpdateConversationMeta(ctx, ex.actorID, ex.conversationID, metaJSON); err != nil {
		t.Fatalf("update conversation meta: %v", err)
	}
	confirm(StageFinalize)
}

// coachTurn drives one HandleMessage turn whose model reply is `reply` and
// returns the conversation row as it stands afterwards.
func coachTurn(t *testing.T, c *Coach, ex *toolExecutor, reply string) string {
	t.Helper()
	ctx := context.Background()
	conv, err := c.store.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	streamer := fakeStreamer{chat: func(string, func(string)) (Turn, error) {
		return Turn{Text: reply}, nil
	}}
	cc := New(c.store, streamer, discardLog())
	cc.HandleMessage(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil),
		ex.conversationID.String(), ex.actorID.String(), conv.WorkspaceID.String(), "wrap it up")
	after, err := c.store.GetConversationForOwner(ctx, ex.conversationID, ex.actorID)
	if err != nil {
		t.Fatalf("re-read conversation: %v", err)
	}
	return after.Status
}

// The finalize prompt's contract is that the report ENDS with the line
// "NO OPEN QUESTIONS" only when nothing is open. Completion keyed on the
// marker as a substring anywhere closes conversations that still list open
// questions — the exact "silently presents incomplete as done" failure.
func TestHandleMessage_CompletionRequiresTheMarkerAsTheFinalLine(t *testing.T) {
	st := newTestStore(t)

	t.Run("marker quoted mid-report with open questions listed does not complete", func(t *testing.T) {
		_, ex := seedConversation(t, st)
		walkToFinalize(t, st, ex)
		c := New(st, fakeStreamer{}, discardLog())
		reply := "完整度报告：\n" +
			"- background: 扎实\n" +
			"- users: 单薄\n\n" +
			"未解决的问题：\n" +
			"1. users 章节缺少目标用户画像\n\n" +
			"（说明：一份干净的报告最后会以单独一行 \"NO OPEN QUESTIONS\" 结束——但前提是没有任何未解决的问题。）"
		if status := coachTurn(t, c, ex, reply); status == "completed" {
			t.Fatalf("conversation completed off a report that still lists open questions and only QUOTES the marker mid-text (status %q)", status)
		}
	})

	t.Run("a report ending with the marker line completes", func(t *testing.T) {
		_, ex := seedConversation(t, st)
		walkToFinalize(t, st, ex)
		c := New(st, fakeStreamer{}, discardLog())
		reply := "完整度报告：\n- background: 扎实\n- users: 扎实\n\nNO OPEN QUESTIONS"
		if status := coachTurn(t, c, ex, reply); status != "completed" {
			t.Fatalf("a report truly ending with the marker line left the conversation %q, want completed", status)
		}
	})

	t.Run("blank lines after the marker line still complete", func(t *testing.T) {
		_, ex := seedConversation(t, st)
		walkToFinalize(t, st, ex)
		c := New(st, fakeStreamer{}, discardLog())
		reply := "完整度报告：\n- background: 扎实\n\nNO OPEN QUESTIONS\n \n\n"
		if status := coachTurn(t, c, ex, reply); status != "completed" {
			t.Fatalf("trailing blank lines hid the marker as the last non-empty line; conversation = %q, want completed", status)
		}
	})
}
