package api

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// SEC-03 / REL-05 (review-3): the auth throttle, email resend cooldown and
// SSE registries are per-process maps, and nothing at runtime ever said so —
// a second replica (or a well-meaning scale-up) silently multiplied every
// one of those budgets with no log line announcing it. Building the handler
// (what startup does once per process in main.go runServe) must state the
// single-replica contract, and a second server process against the same
// database must announce itself loudly. Real Postgres like every other
// store-backed test in this package; skips without DATABASE_URL.

// newSingletonLogServer builds a Server whose logs land in buf, the way the
// cooldown end-to-end test captures the dev-mode mail sender.
func newSingletonLogServer(st *store.Store, logs *bytes.Buffer) *Server {
	return NewServer(st, nil, &config.Config{},
		slog.New(slog.NewTextHandler(logs, nil)), nil, nil)
}

func TestSingletonGuardStatesContractAtStartup(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)

	var logs bytes.Buffer
	s := newSingletonLogServer(st, &logs)
	_ = s.Handler() // startup builds the handler chain exactly once

	out := logs.String()
	if !strings.Contains(out, "single-replica contract") {
		t.Fatalf("startup must state the single-replica contract (per-process throttle/cooldown/turn-lock/SSE state); got logs:\n%s", out)
	}
	if strings.Contains(out, "second replica") {
		t.Fatalf("the sole server against this database must not report a second replica; got logs:\n%s", out)
	}
}

func TestSingletonGuardErrorsWhenASecondServerRuns(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)

	// The incumbent replica: its guard takes the singleton advisory lock on
	// a connection of its own and holds it, exactly like a running process.
	var incumbentLog bytes.Buffer
	incumbent := newSingletonLogServer(st, &incumbentLog)
	_ = incumbent.Handler()

	// The second replica against the same database must say so, loudly.
	var secondLog bytes.Buffer
	second := newSingletonLogServer(st, &secondLog)
	_ = second.Handler()

	out := secondLog.String()
	if !strings.Contains(out, "second replica") || !strings.Contains(out, "level=ERROR") {
		t.Fatalf("a second server process must announce itself with an ERROR log (the review's 'loud instead of silent'); got logs:\n%s", out)
	}
}
