package apiclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Review-4 QUAL-04: a Client took no context, so a cancelled MCP call, or a
// Ctrl-C in the CLI, left its request running until the transport gave up.
// SetContext binds the caller's context to every request the client makes.

// blockingAPI never answers on its own: a request stays open until the client
// leaves it (the safety timer only keeps a regression from hanging the suite).
func blockingAPI(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body) // the server notices a client that left only once the body is read
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
			w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// cancelSoon is a context that is cancelled shortly after it is made.
func cancelSoon(t *testing.T, after time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(after, cancel)
	t.Cleanup(func() { timer.Stop(); cancel() })
	return ctx
}

// wantAbort runs call and requires an error that wraps context.Canceled, well
// before the server would have answered.
func wantAbort(t *testing.T, name string, call func() error) {
	t.Helper()
	start := time.Now()
	err := call()
	if took := time.Since(start); took > 2*time.Second {
		t.Errorf("%s: returned after %v, want an abort shortly after the context was cancelled", name, took)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("%s: err = %v, want it to wrap context.Canceled", name, err)
	}
}

// Every place the client builds a request (do, getBytes, authPost, Guide) must
// carry the context.
func TestCancelledContextAbortsEveryRequestPath(t *testing.T) {
	srv := blockingAPI(t)
	paths := []struct {
		name string
		call func(c *Client) error
	}{
		{"do GET", func(c *Client) error { _, err := c.Whoami(); return err }},
		{"do POST", func(c *Client) error { _, err := c.SyncGithub("p1"); return err }},
		{"getBytes", func(c *Client) error { _, err := c.PrdExport("p1", "markdown", ""); return err }},
		{"Guide", func(c *Client) error { _, err := c.Guide("p1"); return err }},
		{"authPost", func(c *Client) error { return c.RequestLoginCode("a@example.com") }},
	}
	for _, p := range paths {
		t.Run(p.name, func(t *testing.T) {
			c := New(srv.URL)
			c.SetContext(cancelSoon(t, 100*time.Millisecond))
			wantAbort(t, p.name, func() error { return p.call(c) })
		})
	}
}

// A client that was never given a context behaves as before.
func TestClientWithoutAContextStillWorks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"email":"a@example.com"}`)) }))
	t.Cleanup(srv.Close)
	if _, err := New(srv.URL).Whoami(); err != nil {
		t.Fatalf("Whoami without SetContext: %v", err)
	}
}

// The renewal rides the context of the request that triggered it: a Ctrl-C
// while the session is being renewed must not wait for the refresh endpoint.
func TestCancelledContextAbortsARenewal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		if r.URL.Path != "/auth/refresh" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	t.Cleanup(srv.Close)
	f := &file{access: "old", refresh: "ref-1"}
	c := armed(srv.URL, f)
	c.SetContext(cancelSoon(t, 150*time.Millisecond))
	wantAbort(t, "renewal", func() error { _, err := c.Whoami(); return err })
	if f.saved != 0 {
		t.Errorf("a cancelled renewal saved a session %d time(s)", f.saved)
	}
}

// AgentScoped trades the login for the token a tool call runs on. The trade and
// the client it returns both ride the login's context.
func TestAgentScopedRidesTheLoginsContext(t *testing.T) {
	now := time.Now()
	agent := agentTestJWT("u1", "agent", "agent", now, time.Hour)
	login := agentTestJWT("u1", "", "login", now, time.Hour)

	t.Run("the trade", func(t *testing.T) {
		ForgetAgentToken()
		t.Cleanup(ForgetAgentToken)
		c := New(blockingAPI(t).URL)
		c.SetToken(login)
		c.SetContext(cancelSoon(t, 150*time.Millisecond))
		wantAbort(t, "POST /agent/token", func() error { _, err := AgentScoped(c).Whoami(); return err })
	})

	t.Run("the scoped client", func(t *testing.T) {
		ForgetAgentToken()
		t.Cleanup(ForgetAgentToken)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.Copy(io.Discard, r.Body)
			if r.URL.Path == "/agent/token" {
				w.Write([]byte(mintBody(agent)))
				return
			}
			select {
			case <-r.Context().Done():
			case <-time.After(5 * time.Second):
				w.Write([]byte(`{}`))
			}
		}))
		t.Cleanup(srv.Close)
		c := New(srv.URL)
		c.SetToken(login)
		c.SetContext(cancelSoon(t, 300*time.Millisecond))
		wantAbort(t, "GET /me on the agent token", func() error { _, err := AgentScoped(c).Whoami(); return err })
	})
}
