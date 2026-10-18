package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// REL-02: /healthz pinged the pool with the request context and no deadline,
// so a database that stopped answering held the health probe for as long as
// the caller waited (the 15s statement_timeout lives in the database and never
// fires when the statement is not delivered). The platform's health check must
// get its 503 inside its own timeout. Real PostgreSQL behind a relay that
// stalls on demand; DB-gated.
func TestHealthz_AnswersWithinItsBudgetWhenTheDatabaseStalls(t *testing.T) {
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	dsn, err := url.Parse(raw)
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

	tokens := auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour)
	srv := NewServer(st, tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	client := &http.Client{Timeout: 6 * time.Second}
	probe := func() (code int, took time.Duration, err error) {
		start := time.Now()
		resp, err := client.Get(ts.URL + "/healthz")
		took = time.Since(start)
		if err != nil {
			return 0, took, err
		}
		resp.Body.Close()
		return resp.StatusCode, took, nil
	}

	if code, _, err := probe(); err != nil || code != http.StatusOK {
		t.Fatalf("healthz with a live database = %d, %v, want 200", code, err)
	}
	proxy.stall()
	code, took, err := probe()
	if err != nil {
		t.Fatalf("healthz gave no answer within %s while the database was stalled: %v", took.Round(time.Second), err)
	}
	if code != http.StatusServiceUnavailable {
		t.Errorf("healthz with a stalled database = %d, want 503", code)
	}
	if took > 3*time.Second {
		t.Errorf("healthz took %s to say the database is unreachable, want at most 3s (a platform health check gives up first)", took.Round(100*time.Millisecond))
	}
}
