package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// responsesStreamServer answers a Responses POST with the given final
// response.status and hands the captured request JSON back on a channel —
// a real HTTP round trip, no mocks.
func responsesStreamServer(t *testing.T, finalStatus string) (*httptest.Server, <-chan map[string]any) {
	t.Helper()
	reqs := make(chan map[string]any, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode responses request: %v", err)
		}
		reqs <- body
		w.Header().Set("Content-Type", "text/event-stream")
		stream := strings.Join([]string{
			`data: {"type":"response.output_text.delta","delta":"hi"}`,
			``,
			`data: {"type":"response.completed","response":{"usage":{"input_tokens":1,"output_tokens":1},"status":"` + finalStatus + `"}}`,
			``,
		}, "\n")
		w.Write([]byte(stream))
	}))
	t.Cleanup(srv.Close)
	return srv, reqs
}

// AIDEV-06: the Responses streamer sent no max_output_tokens, so per-call
// output cost was uncapped by us — a verbose or looping model could stream
// an unbounded answer once per iteration for up to 12 iterations in a turn.
// Like the Anthropic streamer (maxTokens 4096), the request must carry an
// output cap so a runaway answer is cut off and reported as truncated.
func TestResponsesStreamer_RequestCapsOutputTokens(t *testing.T) {
	srv, reqs := responsesStreamServer(t, "completed")
	s, err := NewResponsesStreamer(srv.URL, "test-token", "gpt-test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.StreamChat(context.Background(), "sys", []Message{{Role: RoleUser, Content: "hi"}}, nil, nil); err != nil {
		t.Fatal(err)
	}
	body := <-reqs
	got, ok := body["max_output_tokens"].(float64)
	if !ok || int(got) != 4096 {
		t.Fatalf("max_output_tokens = %v (present: %v), want 4096 to match the Anthropic streamer's cap", body["max_output_tokens"], ok)
	}
}

// With a cap in place, hitting it is exactly what makes the Responses API
// report status "incomplete" — the F13 truncated marker must keep mapping
// on this provider so a cutoff answer is surfaced, not presented as done.
func TestResponsesStreamer_CapReachedReportsTruncated(t *testing.T) {
	srv, _ := responsesStreamServer(t, "incomplete")
	s, err := NewResponsesStreamer(srv.URL, "test-token", "gpt-test")
	if err != nil {
		t.Fatal(err)
	}
	turn, err := s.StreamChat(context.Background(), "sys", []Message{{Role: RoleUser, Content: "hi"}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !turn.Truncated {
		t.Fatal("turn.Truncated = false for status \"incomplete\", want the cutoff surfaced (F13)")
	}
}
