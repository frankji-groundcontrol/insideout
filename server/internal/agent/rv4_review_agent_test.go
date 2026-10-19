package agent

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"testing/iotest"
)

// rv4-FR-F82: parseResponsesStream returned scanner.Err() as it came. A dropped
// connection mid-stream (io.ErrUnexpectedEOF, a reset) therefore reached
// retryStreamChat and failTurn unclassified: never retried, and counted by the
// breaker as an unknown failure. The Anthropic parser has always filed the same
// failure as transient (classifyStreamReadErr).

// rvAgentCutOff is a stream that delivers its events and is then cut by err.
func rvAgentCutOff(events string, err error) io.Reader {
	return io.MultiReader(strings.NewReader(events), iotest.ErrReader(err))
}

func TestRvAgentParseResponses_ADroppedStreamIsTransient(t *testing.T) {
	cases := map[string]string{
		"after a text delta":             sseData(rText),
		"after a function_call item":     sseData(rCall),
		"in the middle of its arguments": sseData(rCall, rArgs("fc_1", `{"kind":"pro`)),
	}
	for name, events := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := parseResponsesStream(rvAgentCutOff(events, io.ErrUnexpectedEOF), nil, nil)
			if !errors.Is(err, ErrProviderTransient) {
				t.Fatalf("err = %v, want it to wrap ErrProviderTransient so the turn is retried and the breaker counts it as load", err)
			}
			if !strings.Contains(err.Error(), responsesProvider) || strings.Contains(err.Error(), "anthropic") {
				t.Fatalf("err = %q, want it to blame %q and not another provider", err, responsesProvider)
			}
		})
	}
}

// A read that failed because the request was canceled (client gone, idle
// watchdog) is not a provider fault: it must keep reading as a cancel.
func TestRvAgentParseResponses_ACanceledReadStaysACancel(t *testing.T) {
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		_, err := parseResponsesStream(rvAgentCutOff(sseData(rText), cause), nil, nil)
		if !errors.Is(err, cause) || errors.Is(err, ErrProviderTransient) {
			t.Fatalf("read failing with %v gave %v, want that cancellation unclassified", cause, err)
		}
	}
}

// rvAgentStep is one scripted answer; cut declares more body than is sent, so
// the server closes the connection and the client's body read fails with
// io.ErrUnexpectedEOF, as a dropped link does.
type rvAgentStep struct {
	events string
	cut    bool
}

// rvAgentScripted answers its n-th POST with the n-th step (the last repeats)
// and counts the requests it served: a real HTTP round trip, no mocks.
func rvAgentScripted(t *testing.T, steps ...rvAgentStep) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var served atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := min(int(served.Add(1))-1, len(steps)-1)
		w.Header().Set("Content-Type", "text/event-stream")
		if steps[i].cut {
			w.Header().Set("Content-Length", strconv.Itoa(len(steps[i].events)+1024))
		}
		io.WriteString(w, steps[i].events)
		w.(http.Flusher).Flush()
	}))
	t.Cleanup(srv.Close)
	return srv, &served
}

func rvAgentStreamer(t *testing.T, srv *httptest.Server) *ResponsesStreamer {
	t.Helper()
	s, err := NewResponsesStreamer(srv.URL, "test-token", "gpt-test")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

var rvAgentMsgs = []Message{{Role: RoleUser, Content: "hi"}}

// Through the real client: a connection that drops after text reached the
// person is a transient failure, and it is not retried, because a retry would
// replay text they have already read.
func TestRvAgentStreamChat_ADropAfterTextIsTransientAndNotReplayed(t *testing.T) {
	srv, served := rvAgentScripted(t, rvAgentStep{events: sseData(rText), cut: true})
	var deltas []string
	_, err := rvAgentStreamer(t, srv).StreamChat(context.Background(), "sys", rvAgentMsgs, nil, func(d string) { deltas = append(deltas, d) })
	if !errors.Is(err, ErrProviderTransient) {
		t.Fatalf("StreamChat after a dropped connection = %v, want it to wrap ErrProviderTransient", err)
	}
	if n := served.Load(); n != 1 || len(deltas) != 1 {
		t.Fatalf("server saw %d requests and the client %d deltas, want 1 and 1: text already sent is never replayed", n, len(deltas))
	}
}

// A drop before a single delta reached the person is safe to retry, and the
// one retry recovers the turn. (The retry waits out the 2 s transient backoff.)
func TestRvAgentStreamChat_ADropBeforeAnyTextIsRetriedOnce(t *testing.T) {
	srv, served := rvAgentScripted(t,
		rvAgentStep{events: sseData(rCall, rArgs("fc_1", `{"kind":"pro`)), cut: true},
		rvAgentStep{events: sseData(rText, rEnd("response.completed", "completed"))})
	tool := Tool{Name: "get_prd", Description: "reads the prd", Parameters: map[string]any{"type": "object"}}
	turn, err := rvAgentStreamer(t, srv).StreamChat(context.Background(), "sys", rvAgentMsgs, []Tool{tool}, func(string) {})
	if err != nil {
		t.Fatalf("StreamChat = %v, want the second attempt to recover the turn", err)
	}
	if turn.Text != "half an ans" || served.Load() != 2 {
		t.Fatalf("turn.Text = %q after %d requests, want the retry's answer after exactly 2", turn.Text, served.Load())
	}
}
