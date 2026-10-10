package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/presence"
)

// frameWithPrefix reads frames until one starts with prefix.
func frameWithPrefix(t *testing.T, br *bufio.Reader, prefix string) string {
	t.Helper()
	for {
		f, err := readProgressFrame(t, br)
		if err != nil {
			t.Fatalf("waiting for a %q frame: %v", prefix, err)
		}
		if strings.HasPrefix(f, prefix) {
			return f
		}
	}
}

func frameData(t *testing.T, frame string, into any) {
	t.Helper()
	_, data, _ := strings.Cut(frame, "data: ")
	if err := json.Unmarshal([]byte(strings.TrimSpace(data)), into); err != nil {
		t.Fatalf("decode frame %q: %v", frame, err)
	}
}

// snapshotOf reads the next presence snapshot from an open stream.
func (f *streamFixture) snapshotOf(t *testing.T, br *bufio.Reader) []presence.Entry {
	t.Helper()
	var entries []presence.Entry
	frameData(t, frameWithPrefix(t, br, "event: presence"), &entries)
	return entries
}

func (f *streamFixture) listPresence(t *testing.T, token string) []presence.Entry {
	t.Helper()
	resp := doJSON(t, f.ts, token, http.MethodGet, "/api/v1/projects/"+f.proj.ID.String()+"/presence", "")
	defer resp.Body.Close()
	var entries []presence.Entry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		t.Fatalf("decode presence list: %v", err)
	}
	return entries
}

func (f *streamFixture) postCursor(t *testing.T, token, query string) {
	t.Helper()
	resp := doJSON(t, f.ts, token, http.MethodPost, "/api/v1/projects/"+f.proj.ID.String()+"/cursor"+query, `{"x":1,"y":2}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("post cursor%s: status %d, want 204", query, resp.StatusCode)
	}
}

func (f *streamFixture) presencePath(query string) string {
	return "/api/v1/projects/" + f.proj.ID.String() + "/presence/stream" + query
}

// SEC-18: the client id is chosen by the caller and was the registry key as
// is, so any project member could pick another viewer's client id and take
// over that viewer's entry. The key must also carry the authenticated user.
func TestPresence_SameClientIDFromTwoMembersIsTwoSessions(t *testing.T) {
	fx := newStreamFixture(t)
	openSSE(t, fx.ts, fx.tokenA, fx.presencePath("?client=same"))
	b, _ := openSSE(t, fx.ts, fx.tokenB, fx.presencePath("?client=same"))
	entries := fx.snapshotOf(t, b)

	if len(entries) != 2 {
		t.Fatalf("snapshot after both members connected with client=same lists %d entries (%+v), want 2: the second connection took over the first", len(entries), entries)
	}
	users := map[string]bool{entries[0].UserID: true, entries[1].UserID: true}
	if !users[fx.a.ID.String()] || !users[fx.b.ID.String()] {
		t.Fatalf("snapshot users = %+v, want both members", entries)
	}
	if entries[0].SessionID == entries[1].SessionID {
		t.Fatalf("both members share session id %q", entries[0].SessionID)
	}
}

func TestPresence_ADisconnectLeavesAnotherMembersSession(t *testing.T) {
	fx := newStreamFixture(t)
	a, _ := openSSE(t, fx.ts, fx.tokenA, fx.presencePath("?client=same"))
	b, cancelB := openSSE(t, fx.ts, fx.tokenB, fx.presencePath("?client=same"))
	fx.snapshotOf(t, a)
	fx.snapshotOf(t, b)

	cancelB()
	has := func(entries []presence.Entry, userID string) bool {
		for _, e := range entries {
			if e.UserID == userID {
				return true
			}
		}
		return false
	}
	deadline := time.Now().Add(3 * time.Second)
	for has(fx.listPresence(t, fx.tokenA), fx.b.ID.String()) {
		if time.Now().After(deadline) {
			t.Fatal("the second member's session is still listed 3s after it disconnected")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if left := fx.listPresence(t, fx.tokenA); !has(left, fx.a.ID.String()) {
		t.Fatalf("presence after the second member left = %+v, want the first member still listed: its disconnect removed the other's entry", left)
	}
}

func TestPresence_CursorClientIDIsScopedToTheCaller(t *testing.T) {
	fx := newStreamFixture(t)
	b, _ := openSSE(t, fx.ts, fx.tokenB, fx.presencePath("?client=same"))
	mine := fx.snapshotOf(t, b)
	if len(mine) != 1 {
		t.Fatalf("connect snapshot = %+v, want just this viewer", mine)
	}

	fx.postCursor(t, fx.tokenA, "?client=same")
	var ev presence.CursorEvent
	frameData(t, frameWithPrefix(t, b, "event: cursor"), &ev)
	if ev.SessionID == mine[0].SessionID {
		t.Fatalf("the other member's cursor surfaced under this viewer's session id %q", ev.SessionID)
	}
}

// A stream no longer shows the viewer their own pointer: the client used to
// drop it by comparing the session id to its own client id, which stops
// matching once the key carries the user.
func TestPresence_StreamSkipsItsOwnCursorEcho(t *testing.T) {
	fx := newStreamFixture(t)
	a, _ := openSSE(t, fx.ts, fx.tokenA, fx.presencePath("?client=tab-1"))
	b, _ := openSSE(t, fx.ts, fx.tokenB, fx.presencePath("?client=tab-2"))
	fx.snapshotOf(t, a)
	fx.snapshotOf(t, b)

	fx.postCursor(t, fx.tokenA, "?client=tab-1")
	frameWithPrefix(t, b, "event: cursor") // delivered to the other viewer, so the broadcast is done
	// A third session joining is a later change on A's stream: everything A's
	// stream wrote for the pointer comes before the snapshot that lists three.
	openSSE(t, fx.ts, fx.tokenB, fx.presencePath("?client=tab-3"))
	for {
		f, err := readProgressFrame(t, a)
		if err != nil {
			t.Fatalf("read own stream: %v", err)
		}
		if strings.HasPrefix(f, "event: cursor") {
			t.Fatalf("the stream echoed the viewer's own cursor back: %q", f)
		}
		var entries []presence.Entry
		if strings.HasPrefix(f, "event: presence") {
			if frameData(t, f, &entries); len(entries) == 3 {
				return
			}
		}
	}
}

// Callers that send no client id keep today's key, the bare user id; either
// half of the client change can ship first because of it.
func TestPresence_NoClientIDKeepsTheBareUserID(t *testing.T) {
	fx := newStreamFixture(t)
	b, _ := openSSE(t, fx.ts, fx.tokenB, fx.presencePath(""))
	entries := fx.snapshotOf(t, b)
	if len(entries) != 1 || entries[0].SessionID != fx.b.ID.String() {
		t.Fatalf("snapshot = %+v, want one entry keyed by the bare user id", entries)
	}

	a, _ := openSSE(t, fx.ts, fx.tokenA, fx.presencePath(""))
	fx.snapshotOf(t, a)
	fx.postCursor(t, fx.tokenA, "")
	var ev presence.CursorEvent
	frameData(t, frameWithPrefix(t, b, "event: cursor"), &ev)
	if ev.SessionID != fx.a.ID.String() {
		t.Fatalf("cursor without a client id has session id %q, want the bare user id", ev.SessionID)
	}
}

func TestPresence_LongClientIDIsCapped(t *testing.T) {
	fx := newStreamFixture(t)
	long := strings.Repeat("é", 100) // the cap counts characters, not bytes
	a, _ := openSSE(t, fx.ts, fx.tokenA, fx.presencePath("?client="+url.QueryEscape(long)))
	entries := fx.snapshotOf(t, a)
	want := fx.a.ID.String() + ":" + strings.Repeat("é", 64)
	if len(entries) != 1 || entries[0].SessionID != want {
		t.Fatalf("snapshot = %+v, want one entry with session id %q (user id plus the first 64 characters)", entries, want)
	}
}
