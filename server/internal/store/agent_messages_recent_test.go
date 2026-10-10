package store

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
)

// goSideChatTail is what the coach's loadHistory used to do after loading the
// whole conversation: keep user rows and assistant rows that carry text, then
// the last limit of them. ListRecentChatMessages must return exactly this.
func goSideChatTail(all []AgentMessage, limit int) []AgentMessage {
	var out []AgentMessage
	for _, m := range all {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		if m.Role == "assistant" && m.Content == "" {
			continue // a tool-call-only turn, not a text reply
		}
		out = append(out, m)
	}
	if len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

func TestListRecentChatMessages_MatchesGoSideFilterAndTrim(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	user := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, user.ID, "Recent Chat Test WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	// Bodyless idea: ConvertIdea seeds the body as the conversation's
	// first message, and this test must own every row it counts.
	idea, err := st.CreateIdea(ctx, user.ID, ws.ID, "Idea", "")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	_, conv, err := st.ConvertIdea(ctx, user.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}

	// Every row shape the filter must tell apart, oldest first.
	calls := json.RawMessage(`[{"id":"c1","name":"get_prd","arguments":"{}"}]`)
	callID := "c1"
	for _, m := range []struct {
		role, content string
		toolCalls     json.RawMessage
		toolCallID    *string
	}{
		{"user", "u1", nil, nil},
		{"assistant", "a1", nil, nil},
		{"assistant", "", calls, nil}, // a tool-call-only turn
		{"tool", "t1", nil, &callID},
		{"assistant", "a2", nil, nil},
		{"user", "u2", nil, nil},
		{"assistant", "", calls, nil},
		{"tool", "t2", nil, &callID},
		{"assistant", "a3", nil, nil},
		{"user", "u3", nil, nil},
		{"assistant", "", nil, nil}, // the placeholder of a reply still streaming
	} {
		if _, err := st.InsertAgentMessage(ctx, user.ID, conv.ID, m.role, m.content, m.toolCalls, m.toolCallID, nil); err != nil {
			t.Fatalf("insert %s %q: %v", m.role, m.content, err)
		}
	}
	all, err := st.ListAgentMessages(ctx, user.ID, conv.ID)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 11 {
		t.Fatalf("seeded %d rows, want 11", len(all))
	}

	// 1 and 3 trim the tail, 6 is exactly the kept count, 100 exceeds the
	// row count. Contents are spelled out so the oracle is not the only judge.
	kept := []string{"u1", "a1", "a2", "u2", "a3", "u3"}
	for _, limit := range []int{1, 3, 6, 100} {
		got, err := st.ListRecentChatMessages(ctx, user.ID, conv.ID, limit)
		if err != nil {
			t.Fatalf("limit %d: %v", limit, err)
		}
		want := goSideChatTail(all, limit)
		if len(got) != len(want) {
			t.Fatalf("limit %d: got %d rows, the Go-side filter+trim keeps %d", limit, len(got), len(want))
		}
		var contents []string
		for i := range got {
			if got[i].ID != want[i].ID || got[i].Role != want[i].Role || got[i].Content != want[i].Content {
				t.Fatalf("limit %d, row %d: got %s %q, want %s %q", limit, i, got[i].Role, got[i].Content, want[i].Role, want[i].Content)
			}
			contents = append(contents, got[i].Content)
		}
		if wantContents := kept[max(0, len(kept)-limit):]; !slices.Equal(contents, wantContents) {
			t.Fatalf("limit %d: contents = %v, want %v", limit, contents, wantContents)
		}
	}

	// RLS: conversations are owner-only, so another user sees nothing.
	other := mkUser(t, st)
	hidden, err := st.ListRecentChatMessages(ctx, other.ID, conv.ID, 100)
	if err != nil {
		t.Fatalf("non-owner list: %v", err)
	}
	if len(hidden) != 0 {
		t.Fatalf("non-owner saw %d rows of someone else's conversation, want 0", len(hidden))
	}
}
