package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// holdingProvider is a local Anthropic Messages endpoint that streams one
// text delta and then holds the response open until the coach hangs up, like
// a provider that is mid-answer when the server is told to stop.
func holdingProvider(t *testing.T) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/messages") { // the startup /models probe
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, `event: message_start
data: {"type":"message_start","message":{"usage":{"input_tokens":1,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"partial answer"}}

`)
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-time.After(30 * time.Second):
		}
	}))
	t.Cleanup(ts.Close)
	t.Cleanup(ts.CloseClientConnections) // runs first: frees a handler a failed run left parked
	return ts
}

// REL-16: a coach turn still running when shutdown starts used to die with
// the process. The run sat "running" until the reaper's 10-minute sweep and
// the text streamed so far was lost. runServe must cut in-flight requests
// after the drain, so the coach's own terminal-state cleanup has run by the
// time runServe returns. The drain is shortened here; production uses 20s so
// the cleanup still fits the 30s window.
func TestRunServe_ShutdownCutsAnInFlightCoachTurn(t *testing.T) {
	old := shutdownDrain
	shutdownDrain = 300 * time.Millisecond
	t.Cleanup(func() { shutdownDrain = old })
	rig := startShutdownRig(t, holdingProvider(t).URL)

	br := rig.open(t, http.MethodPost, "/api/v1/conversations/"+rig.convID.String()+"/messages", `{"content":"hello coach"}`)
	waitForLine(t, br, "event: delta")

	rig.shutdown()
	select {
	case <-rig.returned:
	case <-time.After(10 * time.Second):
		t.Fatal("runServe still serving 10s after shutdown began: the in-flight coach turn was never cut")
	}

	ctx := context.Background()
	var status string
	if err := rig.st.Pool.QueryRow(ctx, `SELECT status FROM insideout.ai_runs WHERE conversation_id = $1`, rig.convID).Scan(&status); err != nil {
		t.Fatalf("read ai_run: %v", err)
	}
	if status != "failed" {
		t.Fatalf("ai_run status = %q after runServe returned, want %q (an unfinished turn is left for the 10-minute reaper)", status, "failed")
	}
	msgs, err := rig.st.ListAgentMessages(ctx, rig.userID, rig.convID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	var assistant *store.AgentMessage
	for i := range msgs {
		if msgs[i].Role == "assistant" {
			assistant = &msgs[i]
		}
	}
	if assistant == nil {
		t.Fatal("no assistant message was created for the turn")
	}
	if want := "partial answer\n\n[interrupted]"; assistant.Content != want {
		t.Fatalf("assistant message = %q after runServe returned, want the streamed text kept as %q", assistant.Content, want)
	}
}
