package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/api"
	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// rv4-CLOSE-02: `insideout idea convert` sent an empty body, so every idea it
// converted opened a Chinese coach conversation (the server's stored default),
// however the idea was written. The verb now takes --locale and always sends
// one: the flag, else the environment's language, else en.

// convertWire answers a convert with a created PRD and remembers each request
// line (method, path, body) it was sent.
type convertWire struct {
	*httptest.Server
	mu    sync.Mutex
	lines []string
}

func newConvertWire(t *testing.T) *convertWire {
	t.Helper()
	w := &convertWire{}
	w.Server = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		w.mu.Lock()
		w.lines = append(w.lines, r.Method+" "+r.URL.Path+" "+strings.TrimSpace(string(raw)))
		w.mu.Unlock()
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusCreated)
		rw.Write([]byte(`{"prdId":"p1","conversationId":"c1"}`))
	}))
	t.Cleanup(w.Close)
	return w
}

func (w *convertWire) seen() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.lines...)
}

// convertCases are the ways an operator picks a language, with what the
// coach has to end up speaking. The child process gets only the environment
// the case names, so the developer's own LANG cannot decide a result.
var convertCases = []struct {
	name string
	args []string // flags for `idea convert`, before the idea id
	env  []string
	want string
}{
	{"--locale en", []string{"--locale", "en"}, nil, "en"},
	{"--locale zh beats an en environment", []string{"--locale", "zh"}, []string{"LANG=en_US.UTF-8"}, "zh"},
	{"a zh environment", nil, []string{"LANG=zh_CN.UTF-8"}, "zh"},
	{"an en environment", nil, []string{"LANG=en_US.UTF-8"}, "en"},
	{"no language in the environment at all", nil, nil, "en"},
}

func convertArgs(flags []string, ideaID string) []string {
	return append(append([]string{"idea", "convert"}, flags...), ideaID)
}

func TestIdeaConvertSendsTheLocale(t *testing.T) {
	for _, tc := range convertCases {
		t.Run(tc.name, func(t *testing.T) {
			wire := newConvertWire(t)
			writeClientConfig(t, clientconfig.Config{APIBase: wire.URL, Token: "tok"})

			res := runCLI(t, cliEnv(tc.env...), "", convertArgs(tc.args, "i1")...)
			if res.code != 0 {
				t.Fatalf("idea convert: exit %d\n%s", res.code, res.output())
			}
			want := `POST /ideas/i1/convert {"locale":"` + tc.want + `"}`
			if got := wire.seen(); !slices.Equal(got, []string{want}) {
				t.Fatalf("the API saw %q, want exactly [%q]", got, want)
			}
		})
	}
}

// The coach only speaks en and zh, and the server turns every other tag into
// zh, so a locale the verb cannot honour is a usage error and sends nothing.
func TestIdeaConvertRefusesALocaleItCannotCoachIn(t *testing.T) {
	wire := newConvertWire(t)
	writeClientConfig(t, clientconfig.Config{APIBase: wire.URL, Token: "tok"})

	res := runCLI(t, cliEnv(), "", convertArgs([]string{"--locale", "fr"}, "i1")...)
	if res.code != 2 || !strings.Contains(res.stderr, "en or zh") {
		t.Fatalf("idea convert --locale fr: exit %d, want 2 naming en or zh\n%s", res.code, res.output())
	}
	if got := wire.seen(); len(got) != 0 {
		t.Fatalf("a refused locale still reached the API: %q", got)
	}
}

// The same verb against the real handlers over real PostgreSQL: the
// conversation the CLI opens keeps the language it asked for.
// DATABASE_URL-gated.
func TestIdeaConvertStoresTheLocaleOverTheRealServer(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)

	email, password := "convert-"+uuid.NewString()+"@example.org", "pw-for-tests-"+uuid.NewString()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user, err := st.CreateUser(ctx, email, hash, "convert-user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, user.ID, "Convert locale", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	cfg := &config.Config{AccessTTL: time.Hour, RefreshTTL: time.Hour}
	srv := api.NewServer(st, auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour), cfg,
		slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	base := ts.URL + "/api/v1"
	c := apiclient.New(base)
	if err := c.Login(email, password); err != nil {
		t.Fatalf("login: %v", err)
	}
	writeClientConfig(t, clientconfig.Config{APIBase: base, Token: c.Token(), RefreshToken: c.RefreshToken()})

	for _, tc := range convertCases {
		t.Run(tc.name, func(t *testing.T) {
			idea, err := st.CreateIdea(ctx, user.ID, ws.ID, "Convert "+tc.name, "a body")
			if err != nil {
				t.Fatalf("create idea: %v", err)
			}
			res := runCLI(t, cliEnv(tc.env...), "", convertArgs(tc.args, idea.ID.String())...)
			if res.code != 0 {
				t.Fatalf("idea convert: exit %d\n%s", res.code, res.output())
			}
			var converted struct {
				ConversationID string `json:"conversationId"`
			}
			if err := json.Unmarshal([]byte(res.stdout), &converted); err != nil || converted.ConversationID == "" {
				t.Fatalf("decode the convert answer %q: %v", res.stdout, err)
			}
			if got := conversationLocale(t, base, c.Token(), converted.ConversationID); got != tc.want {
				t.Fatalf("the stored conversation locale = %q, want %q", got, tc.want)
			}
		})
	}
}

// conversationLocale reads one conversation back over the API.
func conversationLocale(t *testing.T, base, token, conversationID string) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, base+"/conversations/"+conversationID, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET conversation: %v", err)
	}
	defer resp.Body.Close()
	var view struct {
		Locale string `json:"locale"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&view) != nil {
		t.Fatalf("GET conversation %s: status %d, locale %q", conversationID, resp.StatusCode, view.Locale)
	}
	return view.Locale
}
