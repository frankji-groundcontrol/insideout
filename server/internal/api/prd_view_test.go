package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/agent"
	"github.com/frankji-groundcontrol/insideout/server/internal/audienceview"
	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// TestPrdViewSendsSectionsAsOrderedPicks pins the wire shape the Flutter
// audience page parses (client/test/audience_view_page_test.dart copies it):
// projection.sections is an ordered LIST of {section, why}, one entry per
// section the audience reads. A map keyed by section would lose the reading
// order the projection exists to carry. Skips without DATABASE_URL, same gate
// as server/internal/store/authz_test.go.
func TestPrdViewSendsSectionsAsOrderedPicks(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	ctx := context.Background()

	admin, err := st.CreateUser(ctx, "prd-view-"+uuid.NewString()+"@test.local", "x", "prd-view")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Audience view wire shape", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, admin.ID, ws.ID, "One PRD core", "read by four audiences")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	prd, _, err := st.ConvertIdea(ctx, admin.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}

	tokens := auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour)
	token, err := tokens.MintAccessToken(admin.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	srv := NewServer(st, tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, agent.NewTemplateRoadmapPlanner())
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	for _, audience := range []string{"decision", "management", "delivery", "validation"} {
		resp := doJSON(t, ts, token, http.MethodGet, "/api/v1/prds/"+prd.ID.String()+"/view?audience="+audience, "")
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s view: status %d, want 200; body %s", audience, resp.StatusCode, raw)
		}
		var body struct {
			Projection struct {
				Sections []struct {
					Section string `json:"section"`
					Why     string `json:"why"`
				} `json:"sections"`
			} `json:"projection"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("%s view: projection.sections is not a list of {section, why}: %v; body %s", audience, err, raw)
		}
		want, _ := audienceview.Get(audience)
		if len(body.Projection.Sections) != len(want.Sections) || len(want.Sections) == 0 {
			t.Fatalf("%s view: %d picks, want %d", audience, len(body.Projection.Sections), len(want.Sections))
		}
		for i, pick := range body.Projection.Sections {
			if pick.Section != want.Sections[i].Key || pick.Why == "" {
				t.Errorf("%s view: pick %d = {%q, %q}, want section %q with a why", audience, i, pick.Section, pick.Why, want.Sections[i].Key)
			}
		}
	}
}
