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
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// TestPrdThroughApprovalToProgress is the one planning happy path: approve a
// PRD, build it through handleBuildFromPrd with the template planner, set one
// deadline, mark one leaf done, and read progress back with that leaf counted.
//
// It skips when DATABASE_URL is unset, the same gate as
// server/internal/store/authz_test.go — the tree has to land in real PostgreSQL.
func TestPrdThroughApprovalToProgress(t *testing.T) {
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

	admin, err := st.CreateUser(ctx, "progress-"+uuid.NewString()+"@test.local", "x", "progress-tester")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, admin.ID, "PRD to progress", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, admin.ID, ws.ID, "Ship a thing", "one path")
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

	const wantLeaf = "Walk one user from idea to a done leaf"
	patch(t, ts, token, "/api/v1/prds/"+prd.ID.String(),
		`{"sections":{"goals":"`+wantLeaf+`"}}`, http.StatusOK)

	// Approve the only legal way: draft -> reviewing -> approved
	// (store/prds.go prdTransitions). A commit is the human signature the
	// build handler requires before it will plan.
	postStatus(t, ts, token, prd.ID, "reviewing")
	postStatus(t, ts, token, prd.ID, "approved")
	commit := doJSON(t, ts, token, http.MethodPost, "/api/v1/prds/"+prd.ID.String()+"/commit",
		`{"name":"v1","primaryAudience":"delivery","changeSummary":"ready to build"}`)
	if commit.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(commit.Body)
		commit.Body.Close()
		t.Fatalf("commit: status %d, want 201; body %s", commit.StatusCode, body)
	}
	commit.Body.Close()

	build := doJSON(t, ts, token, http.MethodPost, "/api/v1/prds/"+prd.ID.String()+"/build", "")
	if build.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(build.Body)
		build.Body.Close()
		t.Fatalf("build: status %d, want 200; body %s", build.StatusCode, body)
	}
	var drafted struct {
		ProjectID string          `json:"projectId"`
		Tree      json.RawMessage `json:"tree"`
	}
	if err := json.NewDecoder(build.Body).Decode(&drafted); err != nil {
		build.Body.Close()
		t.Fatalf("decode build: %v", err)
	}
	build.Body.Close()
	if drafted.ProjectID == "" || len(drafted.Tree) == 0 {
		t.Fatalf("build drafted nothing: %+v", drafted)
	}
	confirmBody, err := json.Marshal(map[string]any{"tree": json.RawMessage(drafted.Tree)})
	if err != nil {
		t.Fatalf("marshal confirm: %v", err)
	}
	confirm := doJSON(t, ts, token, http.MethodPost, "/api/v1/prds/"+prd.ID.String()+"/build/confirm", string(confirmBody))
	if confirm.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(confirm.Body)
		confirm.Body.Close()
		t.Fatalf("confirm: status %d, want 201; body %s", confirm.StatusCode, body)
	}
	var built struct {
		ProjectID string `json:"projectId"`
		NodeCount int    `json:"nodeCount"`
	}
	if err := json.NewDecoder(confirm.Body).Decode(&built); err != nil {
		confirm.Body.Close()
		t.Fatalf("decode confirm: %v", err)
	}
	confirm.Body.Close()
	if built.NodeCount == 0 || built.ProjectID == "" {
		t.Fatalf("confirm wrote nothing: %+v", built)
	}

	nodes, err := st.ListRoadmap(ctx, admin.ID, uuid.MustParse(built.ProjectID))
	if err != nil {
		t.Fatalf("list roadmap: %v", err)
	}
	leaf := firstLeaf(nodes, wantLeaf)
	if leaf == nil {
		t.Fatalf("persisted tree has no leaf grounded in the approved PRD (want a leaf title containing %q); got: %s",
			wantLeaf, nodeSummary(nodes))
	}

	deadline := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second).Format(time.RFC3339)
	patch(t, ts, token, "/api/v1/roadmap/"+leaf.ID.String(),
		`{"deadline":"`+deadline+`"}`, http.StatusOK)
	patch(t, ts, token, "/api/v1/roadmap/"+leaf.ID.String(),
		`{"status":"done"}`, http.StatusOK)

	progress := doJSON(t, ts, token, http.MethodGet,
		"/api/v1/projects/"+built.ProjectID+"/roadmap/progress", "")
	if progress.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(progress.Body)
		progress.Body.Close()
		t.Fatalf("progress: status %d, want 200; body %s", progress.StatusCode, body)
	}
	var view struct {
		DoneCount int `json:"doneCount"`
	}
	if err := json.NewDecoder(progress.Body).Decode(&view); err != nil {
		progress.Body.Close()
		t.Fatalf("decode progress: %v", err)
	}
	progress.Body.Close()

	if view.DoneCount != 1 {
		t.Fatalf("progress: want the one done leaf counted, got doneCount %d", view.DoneCount)
	}
}

// postStatus drives one step of the PRD review lifecycle and requires it to
// be accepted.
func postStatus(t *testing.T, ts *httptest.Server, token string, prdID uuid.UUID, status string) {
	t.Helper()
	resp := doJSON(t, ts, token, http.MethodPost, "/api/v1/prds/"+prdID.String()+"/status",
		`{"status":"`+status+`"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST status %s: status %d, want 200; body %s", status, resp.StatusCode, b)
	}
}

// patch sends a JSON body and requires an exact status.
func patch(t *testing.T, ts *httptest.Server, token, path, body string, want int) {
	t.Helper()
	resp := doJSON(t, ts, token, http.MethodPatch, path, body)
	defer resp.Body.Close()
	if resp.StatusCode != want {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("%s %s: status %d, want %d; body %s", http.MethodPatch, path, resp.StatusCode, want, b)
	}
}

// doJSON performs one authenticated JSON request. The caller closes the body.
func doJSON(t *testing.T, ts *httptest.Server, token, method, path, body string) *http.Response {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, ts.URL+path, r)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

// firstLeaf returns the first leaf whose title contains prdText. An empty
// prdText matches any leaf. The pointer addresses the caller's slice.
func firstLeaf(nodes []store.RoadmapNode, prdText string) *store.RoadmapNode {
	parents := map[string]bool{}
	for _, n := range nodes {
		if n.ParentID != nil {
			parents[n.ParentID.String()] = true
		}
	}
	for i, n := range nodes {
		if parents[n.ID.String()] {
			continue
		}
		if prdText == "" || strings.Contains(n.Title, prdText) {
			return &nodes[i]
		}
	}
	return nil
}

func nodeSummary(nodes []store.RoadmapNode) string {
	var b strings.Builder
	for _, n := range nodes {
		b.WriteString("[")
		b.WriteString(n.Title)
		b.WriteString("] ")
	}
	return b.String()
}
