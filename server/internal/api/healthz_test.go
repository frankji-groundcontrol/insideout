package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
)

func TestHealthzAnswersWithoutCredentialsAndFailsWithTheDatabase(t *testing.T) {
	w := newRefreshWorld(t, io.Discard)
	testlock.RequireLocalHost(t, w.st.Pool)
	rec := w.send(http.MethodGet, "/healthz", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("healthz = %d %s, want 200 {\"status\":\"ok\"}", rec.Code, rec.Body.String())
	}

	// The same route over a pool that has been closed: the probe must say so.
	st, err := store.Open(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	st.Close()
	down := NewServer(st, auth.NewTokenIssuer(routeTestSecret, time.Hour), &config.Config{PublicURL: "http://app.test"},
		slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil).Handler()
	rec = httptest.NewRecorder()
	down.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "database unreachable") {
		t.Fatalf("healthz with the database gone = %d %s, want 503 database unreachable", rec.Code, rec.Body.String())
	}
}
