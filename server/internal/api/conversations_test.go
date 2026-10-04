package api

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// TEST-04: the coach SSE endpoint's HTTP-level contract — POST a message,
// receive text/event-stream frames incrementally (frame 1 observable
// before frame 2 is written). DB-gated like the store suite: the
// ownership check runs against real RLS.
func newConversationFixture(t *testing.T) (*store.Store, *store.User, uuid.UUID) {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)

	user, err := st.CreateUser(context.Background(), "sse-"+uuid.NewString()+"@test.local", "x", "sse-tester")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(context.Background(), user.ID, "SSE Test WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(context.Background(), user.ID, ws.ID, "Idea", "content")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	_, conv, err := st.ConvertIdea(context.Background(), user.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea (creates the conversation): %v", err)
	}
	return st, user, conv.ID
}

// stubCoach mimics the real coach's SSE writer (agent/sse.go): frames via
// fmt.Fprintf + explicit Flush. It writes frame 1, signals, then blocks
// until the test releases frame 2 — so the test can prove the frame was
// flushed through the whole middleware stack, not buffered until return.
type stubCoach struct {
	frame1Flushed chan struct{}
	releaseFrame2 chan struct{}
}

func (c stubCoach) HandleMessage(w http.ResponseWriter, r *http.Request, conversationID, userID, workspaceID, content string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	flusher := w.(http.Flusher)
	frame := func(name, data string) {
		io.WriteString(w, "event: "+name+"\ndata: "+data+"\n\n")
		flusher.Flush()
	}
	frame("message_start", `{"id":"m1"}`)
	close(c.frame1Flushed)
	<-c.releaseFrame2
	frame("delta", `{"text":"hello"}`)
	frame("message_end", `{}`)
}

// readFrame reads one SSE frame (up to the blank separator line).
func readFrame(br *bufio.Reader) (string, error) {
	var sb strings.Builder
	for {
		line, err := br.ReadString('\n')
		sb.WriteString(line)
		if err != nil {
			return sb.String(), err
		}
		if line == "\n" {
			return sb.String(), nil
		}
	}
}

func TestSendConversationMessage_StreamsIncrementalFrames(t *testing.T) {
	st, user, convID := newConversationFixture(t)

	tokens := auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour)
	token, err := tokens.MintAccessToken(user.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	coach := stubCoach{frame1Flushed: make(chan struct{}), releaseFrame2: make(chan struct{})}
	srv := NewServer(st, tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), coach, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/conversations/"+convID.String()+"/messages",
		bytes.NewReader([]byte(`{"content":"hello coach"}`)))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("post message: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}

	br := bufio.NewReader(resp.Body)
	// Frame 1 must arrive while the coach is still blocked before frame 2 —
	// if any middleware buffered the reply, this read would hang instead.
	frame1 := make(chan string, 1)
	go func() {
		f, err := readFrame(br)
		if err != nil {
			t.Errorf("read frame 1: %v", err)
		}
		frame1 <- f
	}()
	select {
	case f := <-frame1:
		if !strings.Contains(f, "event: message_start") {
			t.Fatalf("frame 1 = %q, want message_start", f)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("frame 1 never arrived — the response is buffered, not streamed")
	}
	select {
	case <-coach.frame1Flushed:
	default:
		t.Fatal("frame 1 read before the coach flushed it")
	}

	close(coach.releaseFrame2)
	for _, want := range []string{"event: delta", "event: message_end"} {
		f, err := readFrame(br)
		if err != nil {
			t.Fatalf("read %s: %v", want, err)
		}
		if !strings.Contains(f, want) {
			t.Fatalf("frame = %q, want %q", f, want)
		}
	}
}

// The ownership contract at the HTTP layer: a stranger's valid token gets
// 404 (existence not disclosed), not a coach handoff.
func TestSendConversationMessage_StrangerGets404(t *testing.T) {
	st, _, convID := newConversationFixture(t)
	stranger, err := st.CreateUser(context.Background(), "sse-"+uuid.NewString()+"@test.local", "x", "sse-stranger")
	if err != nil {
		t.Fatalf("create stranger: %v", err)
	}

	tokens := auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour)
	token, err := tokens.MintAccessToken(stranger.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	srv := NewServer(st, tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/conversations/"+convID.String()+"/messages",
		bytes.NewReader([]byte(`{"content":"hello"}`)))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("post message: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("stranger status = %d, want 404", resp.StatusCode)
	}
}
