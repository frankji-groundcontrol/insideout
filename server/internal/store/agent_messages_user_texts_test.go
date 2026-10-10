package store

import (
	"context"
	"fmt"
	"slices"
	"testing"
)

// ListUserMessageTexts is what record_fact grounds a quote against: every user
// turn of the conversation, oldest first, however long it has grown. Assistant
// text and tool rows are not the user's words and stay out.
func TestListUserMessageTexts_ReturnsEveryUserTurnOldestFirst(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	user := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, user.ID, "User Texts Test WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	// Bodyless idea: ConvertIdea seeds the body as the conversation's
	// first user turn, and this test must own every turn it asserts.
	idea, err := st.CreateIdea(ctx, user.ID, ws.ID, "Idea", "")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	_, conv, err := st.ConvertIdea(ctx, user.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}

	empty, err := st.ListUserMessageTexts(ctx, user.ID, conv.ID)
	if err != nil || len(empty) != 0 {
		t.Fatalf("a conversation without turns = %v, %v; want no texts", empty, err)
	}

	callID := "c1"
	var want []string
	for i := 0; i < 25; i++ {
		text := fmt.Sprintf("user turn %02d", i)
		want = append(want, text)
		for _, m := range []struct{ role, content string }{
			{"user", text},
			{"assistant", fmt.Sprintf("assistant reply %02d", i)},
		} {
			if _, err := st.InsertAgentMessage(ctx, user.ID, conv.ID, m.role, m.content, nil, nil, nil); err != nil {
				t.Fatalf("insert %s: %v", m.role, err)
			}
		}
		if i%5 == 0 {
			if _, err := st.InsertAgentMessage(ctx, user.ID, conv.ID, "tool", "tool result", nil, &callID, nil); err != nil {
				t.Fatalf("insert tool: %v", err)
			}
		}
	}

	got, err := st.ListUserMessageTexts(ctx, user.ID, conv.ID)
	if err != nil {
		t.Fatalf("list user texts: %v", err)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("user texts = %v, want the 25 user turns oldest first: %v", got, want)
	}

	// RLS: conversations are owner-only, so another user sees nothing.
	other := mkUser(t, st)
	hidden, err := st.ListUserMessageTexts(ctx, other.ID, conv.ID)
	if err != nil {
		t.Fatalf("non-owner list: %v", err)
	}
	if len(hidden) != 0 {
		t.Fatalf("non-owner saw %d of someone else's user turns, want 0", len(hidden))
	}
}
