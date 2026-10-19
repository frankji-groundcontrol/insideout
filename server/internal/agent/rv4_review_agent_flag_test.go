package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
)

// rv4-FR-F83 (also F96, F91): every request that offers tools carried
// parallel_tool_calls:false, and a gateway that does not know the field answers
// 400. providerHTTPError files that as ErrProviderConfig, so the person saw "AI
// provider is misconfigured" on every turn, and nothing the operator could set
// would make the coach work. The streamer now retries once without the field,
// remembers that this provider refuses it, and says so once in the log.

const rvAgentField = "parallel_tool_calls"

// rvAgentGateway stands in for a provider that does not know the field. A
// request carrying it is refused with a 400 body; any other gets a complete
// stream. With always set, every request is refused whatever it carries.
type rvAgentGateway struct {
	mu   sync.Mutex
	sent []bool // one entry per request, in arrival order: did it carry the field?
}

func (g *rvAgentGateway) requests() []bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return slices.Clone(g.sent)
}

func rvAgentGatewayServer(t *testing.T, refusal string, always bool) (*httptest.Server, *rvAgentGateway) {
	t.Helper()
	g := &rvAgentGateway{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_, has := body[rvAgentField]
		g.mu.Lock()
		g.sent = append(g.sent, has)
		g.mu.Unlock()
		if has || always {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, refusal)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sseData(rText, rEnd("response.completed", "completed")))
	}))
	t.Cleanup(srv.Close)
	return srv, g
}

// How a gateway can name the field it refuses: in the message, only in the
// param member (providerHTTPError's error text drops it), or in a bare body.
var rvAgentRefusals = map[string]string{
	"named in the message": `{"error":{"message":"Unknown parameter: 'parallel_tool_calls'.","type":"invalid_request_error","param":"parallel_tool_calls","code":"unknown_parameter"}}`,
	"named only in param":  `{"error":{"message":"Invalid request","type":"invalid_request_error","param":"parallel_tool_calls"}}`,
	"plain text body":      `Unsupported field: parallel_tool_calls`,
}

var rvAgentTool = Tool{Name: "get_prd", Description: "reads the prd", Parameters: map[string]any{"type": "object"}}

var rvAgentEntries = map[string]func(context.Context, *ResponsesStreamer) (Turn, error){
	"StreamChat": func(ctx context.Context, s *ResponsesStreamer) (Turn, error) {
		return s.StreamChat(ctx, "sys", rvAgentMsgs, []Tool{rvAgentTool}, nil)
	},
	"StreamChatForcingTool": func(ctx context.Context, s *ResponsesStreamer) (Turn, error) {
		return s.StreamChatForcingTool(ctx, "sys", rvAgentMsgs, rvAgentTool, nil)
	},
}

// rvAgentCaptureLog points the default logger at a buffer for one test. slog's
// SetDefault also redirects the log package, so both are put back.
func rvAgentCaptureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev, prevOut, prevFlags := slog.Default(), log.Writer(), log.Flags()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() {
		slog.SetDefault(prev)
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})
	return &buf
}

// rvAgentNotices are the log lines that name the field.
func rvAgentNotices(buf *bytes.Buffer) []string {
	var out []string
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.Contains(line, rvAgentField) {
			out = append(out, line)
		}
	}
	return out
}

func TestRvAgentResponses_ARefusedParallelFlagIsRetriedWithoutIt(t *testing.T) {
	for entry, call := range rvAgentEntries {
		for name, refusal := range rvAgentRefusals {
			t.Run(entry+"/"+name, func(t *testing.T) {
				srv, gw := rvAgentGatewayServer(t, refusal, false)
				turn, err := call(context.Background(), rvAgentStreamer(t, srv))
				if err != nil {
					t.Fatalf("%s = %v, want the retry without %s to succeed, not a misconfigured-provider error", entry, err, rvAgentField)
				}
				if turn.Text == "" {
					t.Fatalf("%s returned no text from the retry's complete stream", entry)
				}
				if got := gw.requests(); !slices.Equal(got, []bool{true, false}) {
					t.Fatalf("requests carrying the field = %v, want [true false]: one try with it, one retry without", got)
				}
			})
		}
	}
}

// Once the provider has said no, later turns do not ask again: one request
// each, none carrying the field.
func TestRvAgentResponses_ARefusedFlagIsRememberedAndLoggedOnce(t *testing.T) {
	buf := rvAgentCaptureLog(t)
	srv, gw := rvAgentGatewayServer(t, rvAgentRefusals["named in the message"], false)
	s := rvAgentStreamer(t, srv)
	for i := 1; i <= 3; i++ {
		if _, err := s.StreamChat(context.Background(), "sys", rvAgentMsgs, []Tool{rvAgentTool}, nil); err != nil {
			t.Errorf("turn %d = %v, want it to succeed", i, err)
		}
	}
	if got, want := gw.requests(), []bool{true, false, false, false}; !slices.Equal(got, want) {
		t.Errorf("requests carrying the field = %v, want %v: the refusal is remembered after the first turn", got, want)
	}
	notices := rvAgentNotices(buf)
	if len(notices) != 1 || !strings.Contains(notices[0], "level=WARN") {
		t.Errorf("log lines naming the field = %q, want exactly one, at WARN, so an operator sees it without a flood", notices)
	}
}

// Turns that run side by side and learn the same thing together still produce
// one notice, and the shared memory is safe to race on (this runs under -race).
func TestRvAgentResponses_ConcurrentTurnsLearnItSafelyAndLogOnce(t *testing.T) {
	buf := rvAgentCaptureLog(t)
	srv, _ := rvAgentGatewayServer(t, rvAgentRefusals["named in the message"], false)
	s := rvAgentStreamer(t, srv)
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.StreamChat(context.Background(), "sys", rvAgentMsgs, []Tool{rvAgentTool}, nil); err != nil {
				t.Errorf("concurrent turn = %v, want it to succeed", err)
			}
		}()
	}
	wg.Wait()
	if notices := rvAgentNotices(buf); len(notices) != 1 {
		t.Errorf("log lines naming the field = %q, want exactly one", notices)
	}
}

// A 400 that does not name the field is a different problem: not retried, and
// still the configuration error it was.
func TestRvAgentResponses_AnotherBadRequestIsNotRetried(t *testing.T) {
	refusal := `{"error":{"message":"The model gpt-nope does not exist","type":"invalid_request_error","param":"model"}}`
	srv, gw := rvAgentGatewayServer(t, refusal, true)
	_, err := rvAgentEntries["StreamChat"](context.Background(), rvAgentStreamer(t, srv))
	if !errors.Is(err, ErrProviderConfig) {
		t.Fatalf("err = %v, want ErrProviderConfig", err)
	}
	if got := gw.requests(); !slices.Equal(got, []bool{true}) {
		t.Fatalf("requests carrying the field = %v, want exactly one: nothing to retry without", got)
	}
}

// A provider that refuses the request even without the field is asked twice
// and no more, and the answer is still the configuration error.
func TestRvAgentResponses_ARefusalThatSurvivesDroppingTheFlagStopsAfterOneRetry(t *testing.T) {
	srv, gw := rvAgentGatewayServer(t, rvAgentRefusals["named in the message"], true)
	_, err := rvAgentEntries["StreamChat"](context.Background(), rvAgentStreamer(t, srv))
	if !errors.Is(err, ErrProviderConfig) {
		t.Fatalf("err = %v, want ErrProviderConfig", err)
	}
	if got := gw.requests(); !slices.Equal(got, []bool{true, false}) {
		t.Fatalf("requests carrying the field = %v, want [true false]: one retry, then give up", got)
	}
}
