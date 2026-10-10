package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/google/uuid"
)

// TEST-13: a server started without a roadmap planner must say so (503) on
// the two routes that draft with it, before it reads anything from the store.
// The store is nil here, so a missing guard shows up as a 500 from the
// recovered nil dereference rather than as the 503.
func TestDraftRoutesAnswer503WithoutAPlanner(t *testing.T) {
	tokens := auth.NewTokenIssuer(routeTestSecret, time.Hour)
	token, err := tokens.MintAccessToken(uuid.New())
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	h := NewServer(nil, tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil).Handler()

	for _, path := range []string{
		"/api/v1/prds/" + uuid.NewString() + "/build",
		"/api/v1/roadmap/" + uuid.NewString() + "/expand",
	} {
		t.Run(path, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, path, nil)
			r.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != http.StatusServiceUnavailable {
				t.Fatalf("status %d, want 503 when no planner is configured; body %s", w.Code, w.Body)
			}
		})
	}
}
