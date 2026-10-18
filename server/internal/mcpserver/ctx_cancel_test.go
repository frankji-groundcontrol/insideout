package mcpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// Review-4 QUAL-04: every MCP tool handler receives the call's context, and a
// cancelled call (the client went away, or sent notifications/cancelled) must
// end the HTTP request it made instead of leaving it to run to the transport's
// own limit.

// callToolCtx is callTool with the caller's context.
func callToolCtx(ctx context.Context, t *testing.T, s *server.MCPServer, id int, name, args string) rpcResult {
	t.Helper()
	raw, err := json.Marshal(s.HandleMessage(ctx, json.RawMessage(toolCall(id, name, args))))
	if err != nil {
		t.Fatal(err)
	}
	var r rpcResult
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatalf("tools/call answer %s: %v", raw, err)
	}
	return r
}

func TestToolCallsEndWhenTheirContextIsCancelled(t *testing.T) {
	const slow = 1500 * time.Millisecond
	// A slow API: the sign-in answers at once, everything else after `slow`
	// (or never, once the client has left).
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body) // the server notices a client that left only once the body is read
		if r.URL.Path == "/auth/login" {
			w.Write([]byte(`{"email":"a@example.com","username":"A","accessToken":"tok-session","refreshToken":"ref-session"}`))
			return
		}
		select {
		case <-r.Context().Done():
		case <-time.After(slow):
			w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(api.Close)
	setSessionToken("")
	t.Cleanup(func() { setSessionToken("") })
	s := Build(api.URL)

	cases := []struct{ file, tool, args string }{
		{"product_tools.go", "whoami", `{}`},
		{"admin_tools.go", "signup_mode", `{}`},
		{"auth_tools.go", "auth_login_code_request", `{"email":"a@example.com"}`},
		{"draft_tools.go", "build", `{"prd_id":"p1"}`},
		// The sign-in itself is quick; the /me that confirms it is the slow call.
		{"auth_tools.go (signedIn)", "auth_login", `{"email":"a@example.com","password":"correct horse battery"}`},
	}
	for i, c := range cases {
		ctx, cancel := context.WithCancel(context.Background())
		timer := time.AfterFunc(100*time.Millisecond, cancel)
		start := time.Now()
		res := callToolCtx(ctx, t, s, i+1, c.tool, c.args)
		took := time.Since(start)
		timer.Stop()
		cancel()

		if took > time.Second {
			t.Errorf("%s (%s): returned after %v although its context was cancelled after 100ms", c.tool, c.file, took)
			continue
		}
		if len(res.Result.Content) == 0 || !res.Result.IsError || !strings.Contains(res.Result.Content[0].Text, "canceled") {
			t.Errorf("%s (%s) = %+v, want a tool error saying the request was canceled", c.tool, c.file, res)
		}
	}
}
