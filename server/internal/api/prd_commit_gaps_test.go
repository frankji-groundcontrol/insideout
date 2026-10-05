package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/agent"
	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/readiness"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// TestCommitPersistsEmptySectionGap covers one empty section: readiness
// flags it must_clarify_now, and a commit that sends no unresolved list
// still stores that gap. Skips without DATABASE_URL, same gate as
// server/internal/store/authz_test.go.
func TestCommitPersistsEmptySectionGap(t *testing.T) {
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

	admin, err := st.CreateUser(ctx, "commit-gap-"+uuid.NewString()+"@test.local", "x", "commit-gap")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Commit carries gaps", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, admin.ID, ws.ID, "One empty section", "requirements blank")
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

	filled := map[string]string{}
	for _, key := range store.PrdSectionKeys {
		filled[key] = "written content"
	}
	filled["requirements"] = ""
	body, err := json.Marshal(map[string]any{"sections": filled})
	if err != nil {
		t.Fatalf("marshal sections: %v", err)
	}
	patch(t, ts, token, "/api/v1/prds/"+prd.ID.String(), string(body), http.StatusOK)

	got := readiness.Assess(filled)["delivery"]
	var want string
	for _, item := range got.CarryIntoCommit {
		if strings.HasPrefix(item, "requirements:") {
			want = item
		}
	}
	if want == "" {
		t.Fatalf("readiness did not carry the empty requirements section: %v", got.CarryIntoCommit)
	}
	must := false
	for _, g := range got.Gaps {
		if g.Section == "requirements" && g.Priority == readiness.MustNow {
			must = true
		}
	}
	if !must {
		t.Fatalf("empty requirements is not must_clarify_now: %v", got.Gaps)
	}

	commit := doJSON(t, ts, token, http.MethodPost, "/api/v1/prds/"+prd.ID.String()+"/commit",
		`{"name":"v1","primaryAudience":"delivery","changeSummary":"form a version now"}`)
	defer commit.Body.Close()
	raw, _ := io.ReadAll(commit.Body)
	if commit.StatusCode != http.StatusCreated {
		t.Fatalf("commit: status %d, want 201; body %s", commit.StatusCode, raw)
	}
	var saved struct {
		Unresolved []string `json:"unresolved"`
	}
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatalf("decode commit: %v; body %s", err, raw)
	}
	found := false
	for _, item := range saved.Unresolved {
		if item == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("commit dropped the empty-section gap %q; unresolved %v", want, saved.Unresolved)
	}
}
