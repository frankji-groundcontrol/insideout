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
	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// TestBuildDraftWaitsForConfirm covers the split of POST /prds/{id}/build:
// planning returns a drafted tree and must not write it. The stored roadmap
// stays as it was until POST /prds/{id}/build/confirm persists that tree.
//
// It skips when DATABASE_URL is unset, the same gate as
// server/internal/store/authz_test.go — the roadmap lives in real PostgreSQL.
func TestBuildDraftWaitsForConfirm(t *testing.T) {
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

	admin, err := st.CreateUser(ctx, "build-draft-"+uuid.NewString()+"@test.local", "x", "build-draft")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Draft waits for approval", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, admin.ID, ws.ID, "Drafted roadmap", "plan first")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	prd, _, err := st.ConvertIdea(ctx, admin.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}
	if _, err := st.UpdatePrdStatus(ctx, admin.ID, prd.ID, "reviewing"); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := st.UpdatePrdStatus(ctx, admin.ID, prd.ID, "approved"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, err := st.CreatePrdCommit(ctx, admin.ID, prd.ID, "v1", "delivery", "signed", nil, ""); err != nil {
		t.Fatalf("commit: %v", err)
	}

	tokens := auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour)
	token, err := tokens.MintAccessToken(admin.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	srv := NewServer(st, tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, agent.NewTemplateRoadmapPlanner())
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	drafted := draftBuild(t, ts, token, prd.ID)

	projectID := uuid.MustParse(drafted.ProjectID)
	before, err := st.ListRoadmap(ctx, admin.ID, projectID)
	if err != nil {
		t.Fatalf("list roadmap after draft: %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("build wrote %d roadmap nodes before confirm", len(before))
	}

	confirmBuild(t, ts, token, prd.ID, drafted.Tree)

	after, err := st.ListRoadmap(ctx, admin.ID, projectID)
	if err != nil {
		t.Fatalf("list roadmap after confirm: %v", err)
	}
	if len(after) == 0 {
		t.Fatalf("confirm wrote no roadmap nodes")
	}
}

// draftedBuild is the tree POST /prds/{id}/build returns without writing.
type draftedBuild struct {
	ProjectID string
	Tree      json.RawMessage
}

// draftBuild posts /build and requires a 200 carrying a project and a tree.
func draftBuild(t *testing.T, ts *httptest.Server, token string, prdID uuid.UUID) draftedBuild {
	t.Helper()
	build := doJSON(t, ts, token, http.MethodPost, "/api/v1/prds/"+prdID.String()+"/build", "")
	body, _ := io.ReadAll(build.Body)
	build.Body.Close()
	if build.StatusCode != http.StatusOK {
		t.Fatalf("build: status %d, want 200 (a draft, not a write); body %s", build.StatusCode, body)
	}
	var drafted struct {
		ProjectID string          `json:"projectId"`
		Tree      json.RawMessage `json:"tree"`
	}
	if err := json.Unmarshal(body, &drafted); err != nil {
		t.Fatalf("decode draft: %v; raw %s", err, body)
	}
	if drafted.ProjectID == "" || len(drafted.Tree) == 0 || string(drafted.Tree) == "null" {
		t.Fatalf("draft response missing projectId or tree: %s", body)
	}
	return draftedBuild{ProjectID: drafted.ProjectID, Tree: drafted.Tree}
}

// confirmBuild posts /build/confirm with the drafted tree and requires 201.
func confirmBuild(t *testing.T, ts *httptest.Server, token string, prdID uuid.UUID, tree json.RawMessage) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"tree": json.RawMessage(tree)})
	if err != nil {
		t.Fatalf("marshal confirm: %v", err)
	}
	confirm := doJSON(t, ts, token, http.MethodPost, "/api/v1/prds/"+prdID.String()+"/build/confirm", string(body))
	raw, _ := io.ReadAll(confirm.Body)
	confirm.Body.Close()
	if confirm.StatusCode != http.StatusCreated {
		t.Fatalf("confirm: status %d, want 201; body %s", confirm.StatusCode, raw)
	}
}
