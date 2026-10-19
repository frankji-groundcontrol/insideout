package mcpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// rv4-CLOSE-02: the idea_convert tool sent an empty body, so every idea an
// agent converted opened a Chinese coach conversation (the server's stored
// default), however the idea was written. The tool now takes a locale and
// always sends one: the argument, else the process language, else en.

// convertLocaleWire answers a convert with a created PRD and remembers each
// request line (method, path, body) it was sent.
type convertLocaleWire struct {
	*httptest.Server
	mu    sync.Mutex
	lines []string
}

func newConvertLocaleWire(t *testing.T) *convertLocaleWire {
	t.Helper()
	w := &convertLocaleWire{}
	w.Server = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		w.mu.Lock()
		w.lines = append(w.lines, r.Method+" "+r.URL.Path+" "+strings.TrimSpace(string(raw)))
		w.mu.Unlock()
		rw.WriteHeader(http.StatusCreated)
		rw.Write([]byte(`{"prdId":"p1","conversationId":"c1"}`))
	}))
	t.Cleanup(w.Close)
	return w
}

func (w *convertLocaleWire) seen() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.lines...)
}

// pinLocaleEnv gives the process the language a test names (and none of the
// machine's own).
func pinLocaleEnv(t *testing.T, lang string) {
	t.Helper()
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", lang)
}

func TestIdeaConvertToolSendsALocale(t *testing.T) {
	for _, tc := range []struct{ name, args, lang, wantBody string }{
		{"locale en", `{"idea_id":"i1","locale":"en"}`, "", `{"locale":"en"}`},
		{"locale zh beats an en environment", `{"idea_id":"i1","locale":"zh"}`, "en_US.UTF-8", `{"locale":"zh"}`},
		{"no locale follows a zh environment", `{"idea_id":"i1"}`, "zh_CN.UTF-8", `{"locale":"zh"}`},
		{"no locale and no environment is en", `{"idea_id":"i1"}`, "", `{"locale":"en"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pinLocaleEnv(t, tc.lang)
			wire := newConvertLocaleWire(t)

			res := drive(t, wire.URL, []string{toolCall(1, "idea_convert", tc.args)})
			if res[1].Result.IsError {
				t.Fatalf("idea_convert failed: %+v", res[1])
			}
			want := "POST /ideas/i1/convert " + tc.wantBody
			if got := wire.seen(); len(got) != 1 || got[0] != want {
				t.Fatalf("the API saw %q, want exactly [%q]", got, want)
			}
		})
	}
}

// The coach only speaks en and zh, and the server turns every other tag into
// zh, so a locale the tool cannot honour is an error, and nothing is sent.
func TestIdeaConvertToolRefusesALocaleItCannotCoachIn(t *testing.T) {
	pinLocaleEnv(t, "")
	wire := newConvertLocaleWire(t)

	res := drive(t, wire.URL, []string{toolCall(1, "idea_convert", `{"idea_id":"i1","locale":"fr"}`)})
	text := ""
	if c := res[1].Result.Content; len(c) > 0 {
		text = c[0].Text
	}
	if !res[1].Result.IsError || !strings.Contains(text, "en or zh") {
		t.Fatalf("idea_convert with locale fr = error %v %q, want a tool error naming en or zh", res[1].Result.IsError, text)
	}
	if got := wire.seen(); len(got) != 0 {
		t.Fatalf("a refused locale still reached the API: %q", got)
	}
}

// An agent learns the argument from tools/list alone, so the schema has to
// declare locale and say which values it takes.
func TestIdeaConvertToolSchemaDeclaresLocale(t *testing.T) {
	msg := Build("http://unused.example.org").HandleMessage(context.Background(),
		json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				InputSchema struct {
					Properties map[string]struct {
						Description string `json:"description"`
					} `json:"properties"`
				} `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("tools/list answer %s: %v", raw, err)
	}
	for _, tool := range out.Result.Tools {
		if tool.Name != "idea_convert" {
			continue
		}
		prop, ok := tool.InputSchema.Properties["locale"]
		if !ok {
			t.Fatalf("idea_convert declares no locale argument (its handler sends one)")
		}
		for _, want := range []string{"en", "zh"} {
			if !strings.Contains(prop.Description, want) {
				t.Errorf("idea_convert locale description %q does not name %q", prop.Description, want)
			}
		}
		return
	}
	t.Fatal("idea_convert is not offered to a process with no credential")
}
