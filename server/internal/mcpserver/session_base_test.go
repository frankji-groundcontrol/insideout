package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/server"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// callTool runs one tools/call against an already-built server, so a test can
// change the world between two calls of the same MCP process.
func callTool(t *testing.T, s *server.MCPServer, id int, name, args string) rpcResult {
	t.Helper()
	raw, err := json.Marshal(s.HandleMessage(context.Background(), json.RawMessage(toolCall(id, name, args))))
	if err != nil {
		t.Fatal(err)
	}
	var r rpcResult
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatalf("tools/call answer %s: %v", raw, err)
	}
	return r
}

// sessionAPI is an API that signs anyone in as "tok-session" and records every
// authorization header it is shown.
func sessionAPI(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var auths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		auths = append(auths, r.Header.Get("Authorization"))
		mu.Unlock()
		if r.URL.Path == "/auth/login" {
			w.Write([]byte(`{"email":"a@example.com","username":"A","accessToken":"tok-session","refreshToken":"ref-session"}`))
			return
		}
		w.Write([]byte(`{"email":"a@example.com"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), auths...)
	}
}

// Review-4 SEC-05: a token obtained through the auth tools belongs to the
// server it was obtained from. The base is settled when the server is built, so
// a later change to the saved login's base (another terminal ran
// `insideout login --api ...`) cannot redirect a running MCP process's session
// to a different server.
func TestSessionTokenStaysOnTheServerItCameFrom(t *testing.T) {
	pinConfigHome(t)
	t.Cleanup(func() { setSessionToken("") })
	a, authsA := sessionAPI(t)
	b, authsB := sessionAPI(t)
	saveConfig(t, clientconfig.Config{APIBase: a.URL})
	s := Build("") // the MCP binary passes no --api: the base is the saved login's

	if r := callTool(t, s, 1, "auth_login", `{"email":"a@example.com","password":"pw123456"}`); r.Result.IsError {
		t.Fatalf("auth_login failed: %+v", r)
	}
	saveConfig(t, clientconfig.Config{APIBase: b.URL, Token: "tok-b"})
	callTool(t, s, 2, "whoami", `{}`)

	for _, got := range authsB() {
		if strings.Contains(got, "tok-session") {
			t.Errorf("the session token from server A was sent to server B (Authorization %q)", got)
		}
	}
	if got := authsA(); len(got) < 3 || got[len(got)-1] != "Bearer tok-session" {
		t.Errorf("whoami must keep riding the session on the server that issued it, A saw %q", got)
	}
}

// The saved login's token is never offered to a server the process was pointed
// at explicitly (--api).
func TestBuildDoesNotSendAnotherServersSavedToken(t *testing.T) {
	pinConfigHome(t)
	b, seenB := recordingAPI(t)
	saveConfig(t, clientconfig.Config{APIBase: "https://a.example/api/v1", Token: "tok-a"})

	res := callTool(t, Build(b.URL), 1, "whoami", `{}`)

	if res.Result.IsError {
		t.Fatalf("whoami failed: %+v", res)
	}
	if got := seenB(); len(got) != 1 || got[0] != "GET /me " {
		t.Fatalf("server B saw %q, want one unauthenticated GET /me", got)
	}
}
