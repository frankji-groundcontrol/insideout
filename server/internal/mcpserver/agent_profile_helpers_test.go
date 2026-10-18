package mcpserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// Shared vocabulary and fakes for the agent profile tests (agent_profile_test.go).

// humanOnlyTools are the tools an agent-scope token cannot use on the server, or
// that put a password in the conversation.
var humanOnlyTools = []string{
	"commit", "proposal_decide", "signup_mode", "invite_create", "invite_list", "invite_revoke",
	"idea_delete", "roadmap_delete", "auth_register", "auth_login", "auth_forgot_password", "auth_reset_password",
}

// agentTools must survive the profile: the agent's own vocabulary, the edits it
// makes for the user and the passwordless sign-in that can still start a session.
var agentTools = []string{
	"whoami", "workspaces", "projects", "prd", "prd_update", "idea_create", "roadmap_add", "agent_context",
	"checkpoint", "propose", "auth_login_code_request", "auth_login_code_verify",
	"auth_magic_link_request", "auth_magic_link_consume", "auth_logout",
}

const exampleAPI = "http://api.example.com"

// fakeJWT shapes a token like the ones the server issues (sub, iat, exp, an
// optional scope). The MCP side never verifies a signature, so none is real;
// tag keeps two tokens minted in the same second apart.
func fakeJWT(scope, tag string, issued time.Time, ttl time.Duration) string {
	claims := map[string]any{
		"sub": "4f0d8c5e-5d3b-4f6e-9a55-0d9a2f1c7b11", "tag": tag,
		"iat": issued.Unix(), "exp": issued.Add(ttl).Unix(),
	}
	if scope != "" {
		claims["scope"] = scope
	}
	payload, _ := json.Marshal(claims)
	enc := base64.RawURLEncoding
	return enc.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)) + "." + enc.EncodeToString(payload) + "." + enc.EncodeToString([]byte("sig"))
}

func humanJWT() string { return fakeJWT("", "human", time.Now(), 15*time.Minute) }

// mintReply is what POST /agent/token answers.
type mintReply struct {
	token   string
	expires time.Time
}

func agentGrant(tag string, issued time.Time, ttl time.Duration) mintReply {
	return mintReply{token: fakeJWT("agent", tag, issued, ttl), expires: issued.Add(ttl)}
}

func (m mintReply) body() string {
	return fmt.Sprintf(`{"accessToken":%q,"expiresAt":%q,"scope":"agent"}`, m.token, m.expires.UTC().Format(time.RFC3339))
}

// mintsAs answers the nth POST /agent/token with the nth reply (the last one
// repeats) and leaves every other request to the caller.
func mintsAs(replies ...mintReply) func(http.ResponseWriter, *http.Request, int) bool {
	return func(w http.ResponseWriter, r *http.Request, nth int) bool {
		if r.Method != http.MethodPost || r.URL.Path != "/agent/token" {
			return false
		}
		w.Write([]byte(replies[min(nth, len(replies))-1].body()))
		return true
	}
}

// routedAPI records every request as "METHOD path Authorization". handle may
// answer a request (true; nth counts the requests so far to the same METHOD
// path); anything it declines gets {"email":"a@example.com"}.
func routedAPI(t *testing.T, handle func(w http.ResponseWriter, r *http.Request, nth int) bool) (*httptest.Server, func(pairs ...string) []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	count := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := r.Method + " " + r.URL.Path
		mu.Lock()
		seen = append(seen, route+" "+r.Header.Get("Authorization"))
		count[route]++
		nth := count[route]
		mu.Unlock()
		if handle == nil || !handle(w, r, nth) {
			w.Write([]byte(`{"email":"a@example.com"}`))
		}
	}))
	t.Cleanup(srv.Close)
	// The recorded list, with the given token/label pairs substituted so a
	// failure reads "GET /me Bearer <human>" instead of a wall of base64.
	return srv, func(pairs ...string) []string {
		mu.Lock()
		defer mu.Unlock()
		r := strings.NewReplacer(pairs...)
		out := make([]string, len(seen))
		for i, s := range seen {
			out[i] = r.Replace(s)
		}
		return out
	}
}

func offeredTools(t *testing.T, s *server.MCPServer) []string {
	t.Helper()
	raw, err := json.Marshal(s.HandleMessage(context.Background(), json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)))
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("tools/list answer %s: %v", raw, err)
	}
	var names []string
	for _, tool := range out.Result.Tools {
		names = append(names, tool.Name)
	}
	return names
}

func stillOffered(names []string) []string {
	var got []string
	for _, n := range humanOnlyTools {
		if slices.Contains(names, n) {
			got = append(got, n)
		}
	}
	return got
}

func wantSeen(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("the API saw\n  %s\nwant\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}
