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

// TestBuildFromPrdDraftsUnapproved covers the draft/confirm split on a PRD
// that is still a draft. HEAD's handleBuildFromPrd planned regardless of
// status, and the split keeps that: POST /prds/{id}/build returns the drafted
// tree and writes nothing until confirm.
//
// It skips when DATABASE_URL is unset, the same gate as
// server/internal/store/authz_test.go — the roadmap lives in real PostgreSQL.
func TestBuildFromPrdDraftsUnapproved(t *testing.T) {
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

	admin, err := st.CreateUser(ctx, "build-status-"+uuid.NewString()+"@test.local", "x", "build-status")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Draft needs no approval", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, admin.ID, ws.ID, "Unapproved version", "still a draft")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	prd, _, err := st.ConvertIdea(ctx, admin.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
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
		t.Fatalf("draft of an unapproved PRD wrote %d roadmap nodes", len(nodes))
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
