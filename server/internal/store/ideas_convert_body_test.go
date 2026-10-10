package store

import (
	"context"
	"testing"
)

// FEAT-02: PRODUCT.md's canonical journey says conversion "preserves the
// original title and body as coaching context". The title reaches the PRD
// via insertPrd, but the body used to be scanned and dropped: the coaching
// conversation started empty, so the coach's first question ignored
// everything the user had already written — it survived only on the inbox
// card. ConvertIdea must seed the conversation with the idea body as its
// first user message, verbatim.
func TestConvertIdea_PreservesBodyAsFirstMessage(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	author := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, author.ID, "Convert Body WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	const body = "line one: the problem\nline two: who it serves\nline three: why now"
	idea, err := st.CreateIdea(ctx, author.ID, ws.ID, "Idea", body)
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}

	prd, conv, err := st.ConvertIdea(ctx, author.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}

	msgs, err := st.ListAgentMessages(ctx, author.ID, conv.ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatalf("conversation %s has no messages; the idea body never reached the coach", conv.ID)
	}
	if msgs[0].Role != "user" {
		t.Fatalf("first message role = %q, want user", msgs[0].Role)
	}
	if msgs[0].Content != body {
		t.Fatalf("first user message = %q, want the idea body verbatim:\n%q", msgs[0].Content, body)
	}

	// No regression: the idea still ends converted with prd_id set.
	got, err := st.GetIdeaForMember(ctx, idea.ID, author.ID)
	if err != nil {
		t.Fatalf("get idea: %v", err)
	}
	if got.Status != "converted" {
		t.Fatalf("idea status = %q, want converted", got.Status)
	}
	if got.PrdID == nil || *got.PrdID != prd.ID {
		t.Fatalf("idea prd_id = %v, want the converted PRD %s", got.PrdID, prd.ID)
	}
}
