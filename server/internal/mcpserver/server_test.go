package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// End-to-end over the real stdio protocol: JSON-RPC lines in, responses
// out, against an httptest /api/v1. This is the layer parity lives at —
// the auth tools must sign the process in and every later tool must ride
// that in-process session.

type rpcResult struct {
	ID     int `json:"id"`
	Result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	} `json:"result"`
}

func drive(t *testing.T, api string, lines []string) map[int]rpcResult {
	t.Helper()
	srv := server.NewStdioServer(Build(api))
	pr, pw := io.Pipe()
	outR, outW := io.Pipe()
	done := make(chan error, 1)
	go func() { done <- srv.Listen(context.Background(), pr, outW) }()

	go func() {
		for _, l := range lines {
			pw.Write([]byte(l + "\n"))
		}
		pw.Close()
	}()

	results := make(map[int]rpcResult)
	want := 0
	for _, l := range lines {
		if strings.Contains(l, `"id"`) {
			want++
		}
	}
	deadline := time.After(10 * time.Second)
	sc := bufio.NewScanner(outR)
	for len(results) < want {
		select {
		case <-deadline:
			t.Fatalf("timed out with %d/%d responses", len(results), want)
		default:
		}
		if !sc.Scan() {
			break
		}
		line := sc.Text()
		if !strings.Contains(line, `"id"`) {
			continue // notification ack
		}
		var r rpcResult
		if err := json.Unmarshal([]byte(line), &r); err != nil || r.ID == 0 {
			continue
		}
		results[r.ID] = r
	}
	pw.Close()
	outW.Close()
	<-done
	return results
}

func toolCall(id int, name, args string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"%s","arguments":%s}}`, id, name, args)
}

func TestAuthToolsSignInProcessAndWhoamiRidesSession(t *testing.T) {
	var gotBearer string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBearer = r.Header.Get("Authorization")
		switch r.URL.Path {
		case "/auth/login-code":
			w.Write([]byte(`{"ok":true}`))
		case "/auth/login-code/verify":
			w.Write([]byte(`{"email":"a@b.c","username":"A","accessToken":"tok-mcp"}`))
		case "/me":
			if gotBearer == "Bearer tok-mcp" {
				w.Write([]byte(`{"email":"a@b.c","username":"A"}`))
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"no token"}`))
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	t.Cleanup(api.Close)

	res := drive(t, api.URL, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		toolCall(2, "auth_login_code_request", `{"email":"a@b.c"}`),
		toolCall(3, "auth_login_code_verify", `{"email":"a@b.c","code":"123456"}`),
		toolCall(4, "whoami", `{}`),
	})

	if r := res[2]; !strings.Contains(r.Result.Content[0].Text, "code sent if the account exists") {
		t.Errorf("request tool text = %q", r.Result.Content[0].Text)
	}
	signed := res[3].Result.Content[0].Text
	if !strings.Contains(signed, "signed in as a@b.c") || strings.Contains(signed, "tok-mcp") {
		t.Errorf("verify must report the user and never the token, got %q", signed)
	}
	// whoami succeeded (not a tool error) — it rode the in-process session.
	if res[4].Result.IsError || !strings.Contains(res[4].Result.Content[0].Text, "a@b.c") {
		t.Errorf("whoami after login = %+v", res[4])
	}

	// A fresh process (new Build) has no session: whoami falls back to
	// no token and surfaces the API's 401 as a tool error. (Clear the
	// package-global first — a real second process starts cold.)
	setSessionToken("")
	res2 := drive(t, api.URL, []string{
		toolCall(9, "whoami", `{}`),
	})
	if !res2[9].Result.IsError {
		t.Errorf("whoami without a session must be an error, got %+v", res2[9])
	}
}

func TestAuthLogoutClearsSession(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			w.Write([]byte(`{"email":"a@b.c","username":"A","accessToken":"tok-2"}`))
		case "/me":
			if r.Header.Get("Authorization") == "Bearer tok-2" {
				w.Write([]byte(`{"email":"a@b.c","username":"A"}`))
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"no token"}`))
		}
	}))
	t.Cleanup(api.Close)

	res := drive(t, api.URL, []string{
		toolCall(1, "auth_login", `{"email":"a@b.c","password":"pw123456"}`),
		toolCall(2, "auth_logout", `{}`),
		toolCall(3, "whoami", `{}`),
	})
	if res[1].Result.IsError {
		t.Fatalf("auth_login failed: %+v", res[1])
	}
	if res[2].Result.IsError {
		t.Fatalf("auth_logout failed: %+v", res[2])
	}
	if !res[3].Result.IsError {
		t.Errorf("whoami after logout must be an error (session cleared), got %+v", res[3])
	}
}
