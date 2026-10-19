package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// rv4-FR-F129: the two confirm routes only write the tree the caller already
// holds, yet both sat in deadlineExempt, so a database that stopped answering
// held them until the caller gave up. Real PostgreSQL behind a relay that
// stalls on demand (the technique of the /healthz test: statement_timeout lives
// in the server and cannot fire on a statement that is never delivered, so only
// the request deadline can end the wait). The plain route is the control: it
// proves the stall is real and that a bounded route answers. DB-gated.
func TestRvGuardsConfirmRoutesAreBoundedWhenTheDatabaseStalls(t *testing.T) {
	f := newGateFixture(t) // seeds the rows over a direct pool; skips without DATABASE_URL
	dsn, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	proxy := newStallProxy(t, dsn.Host)
	dsn.Host = proxy.addr()
	st, err := store.Open(context.Background(), dsn.String())
	if err != nil {
		t.Fatalf("open store through the relay: %v", err)
	}
	t.Cleanup(st.Close)

	old := requestDeadline
	requestDeadline = 300 * time.Millisecond
	t.Cleanup(func() { requestDeadline = old })

	srv := NewServer(st, f.tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	// Last in, first out: the relay's sockets close before ts.Close waits for a
	// handler still stuck on the dead connection, and before st.Close.
	t.Cleanup(proxy.close)

	client := &http.Client{Timeout: 4 * time.Second}
	do := func(method, path, body string) (code int, took time.Duration, err error) {
		req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatalf("build request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+f.token)
		start := time.Now()
		resp, err := client.Do(req)
		took = time.Since(start)
		if err != nil {
			return 0, took, err
		}
		resp.Body.Close()
		return resp.StatusCode, took, nil
	}
	cases := []struct {
		name, method, path, body string
		live                     int // the answer while the database is healthy
	}{
		{"control: a plain route", http.MethodGet, "/api/v1/workspaces", "", http.StatusOK},
		{"build confirm", http.MethodPost, "/api/v1/prds/" + f.prdID.String() + "/build/confirm", `{"tree":{"title":"Root"}}`, http.StatusConflict},
		{"expand confirm", http.MethodPost, "/api/v1/roadmap/" + f.nodeID.String() + "/expand/confirm", `{"children":[{"title":"Child"}]}`, http.StatusCreated},
	}
	for _, c := range cases { // a malformed request would also answer fast and prove nothing
		if code, _, err := do(c.method, c.path, c.body); err != nil || code != c.live {
			t.Fatalf("%s with a live database = %d, %v, want %d", c.name, code, err, c.live)
		}
	}

	proxy.stall()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, took, err := do(c.method, c.path, c.body)
			if err != nil {
				t.Fatalf("no answer within %s while the database was stalled: %v", took.Round(100*time.Millisecond), err)
			}
			if code < 500 || code > 599 {
				t.Errorf("status %d while the database was stalled, want a 5xx", code)
			}
			if took > 2*time.Second {
				t.Errorf("took %s to give up on the stalled database, want about the %s request deadline", took.Round(100*time.Millisecond), requestDeadline)
			}
		})
	}
}
