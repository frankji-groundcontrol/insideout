package api

import (
	"context"
	"io"
	"log/slog"
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

// TestBuildFromPrdDraftsWithoutCommit covers the draft/confirm split on a PRD
// that has no commit and is still a draft — the same PRD HEAD's
// handleBuildFromPrd would plan. POST /prds/{id}/build returns the drafted
// tree and writes nothing; the stored roadmap stays empty until confirm.
//
// It skips when DATABASE_URL is unset, the same gate as
// server/internal/store/authz_test.go — the roadmap lives in real PostgreSQL.
func TestBuildFromPrdDraftsWithoutCommit(t *testing.T) {
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

	admin, err := st.CreateUser(ctx, "build-commit-"+uuid.NewString()+"@test.local", "x", "build-commit")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Draft needs no commit", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, admin.ID, ws.ID, "Unsigned version", "no signature yet")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	prd, _, err := st.ConvertIdea(ctx, admin.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}
	commits, err := st.ListPrdCommits(ctx, admin.ID, prd.ID)
	if err != nil {
		t.Fatalf("list commits: %v", err)
	}
	if len(commits) != 0 {
		t.Fatalf("fresh PRD already has %d commits", len(commits))
	}
	if prd.Status != "draft" {
		t.Fatalf("fresh PRD status = %q, want draft", prd.Status)
	}

	tokens := auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour)
	token, err := tokens.MintAccessToken(admin.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	srv := NewServer(st, tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, agent.NewTemplateRoadmapPlanner())
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	project, err := st.EnsureProjectForPrd(ctx, admin.ID, prd.ID)
	if err != nil {
		t.Fatalf("ensure project: %v", err)
	}

	drafted := draftBuild(t, ts, token, prd.ID)
	nodes, err := st.ListRoadmap(ctx, admin.ID, project.ID)
	if err != nil {
		t.Fatalf("list roadmap after draft: %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("draft of a PRD with no commit wrote %d roadmap nodes", len(nodes))
	}

	confirmBuild(t, ts, token, prd.ID, drafted.Tree)
	after, err := st.ListRoadmap(ctx, admin.ID, project.ID)
	if err != nil {
		t.Fatalf("list roadmap after confirm: %v", err)
	}
	if len(after) == 0 {
		t.Fatalf("confirm wrote no roadmap nodes")
	}
}
