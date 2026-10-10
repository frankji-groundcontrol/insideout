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

// TestCommitListReturnsDecisionNoteAndSectionDiff pins what the Flutter commit
// card reads back: the decision note a commit was given, and a diff that names
// every changed section as {"change": ...} (prdcommit.Diff), not only counts.
// Nothing else covered the note's round trip. Skips without DATABASE_URL, same
// gate as prd_commit_gaps_test.go.
func TestCommitListReturnsDecisionNoteAndSectionDiff(t *testing.T) {
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

	admin, err := st.CreateUser(ctx, "commit-note-"+uuid.NewString()+"@test.local", "x", "commit-note")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, admin.ID, "Commit records its diff", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, admin.ID, ws.ID, "Two commits", "second one changes a section")
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

	path := "/api/v1/prds/" + prd.ID.String()
	const note = "Go B2B first; the consumer plan is the alternative not taken"
	patch(t, ts, token, path, `{"sections":{"background":"first draft","goals":"first goals"}}`, http.StatusOK)
	mustCommit(t, ts, token, path, `{"name":"v1","primaryAudience":"delivery"}`)
	patch(t, ts, token, path, `{"sections":{"background":"a longer, revised draft","goals":"first goals"}}`, http.StatusOK)
	mustCommit(t, ts, token, path, `{"name":"v2","primaryAudience":"delivery","changeSummary":"revised background","decisionNote":"`+note+`"}`)

	resp := doJSON(t, ts, token, http.MethodGet, path+"/commits", "")
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list commits: status %d; body %s", resp.StatusCode, raw)
	}
	type change struct {
		Change string `json:"change"`
	}
	var list []struct {
		Name         string `json:"name"`
		Summary      string `json:"changeSummary"`
		DecisionNote string `json:"decisionNote"`
		Diff         struct {
			Counts   map[string]int    `json:"counts"`
			Sections map[string]change `json:"sections"`
		} `json:"diff"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatalf("decode commits: %v; body %s", err, raw)
	}
	byName := map[string]int{}
	for i, c := range list {
		byName[c.Name] = i
	}
	i1, ok1 := byName["v1"]
	i2, ok2 := byName["v2"]
	if !ok1 || !ok2 {
		t.Fatalf("commits v1/v2 missing from %s", raw)
	}
	v1, v2 := list[i1], list[i2]

	if v1.DecisionNote != "" {
		t.Errorf("v1 decisionNote = %q, want empty", v1.DecisionNote)
	}
	if v1.Diff.Sections["background"].Change != "added" {
		t.Errorf("v1 background change = %q, want added (nothing precedes it)", v1.Diff.Sections["background"].Change)
	}
	if v2.DecisionNote != note || v2.Summary != "revised background" {
		t.Errorf("v2 decisionNote/summary = %q / %q, want the ones it was committed with", v2.DecisionNote, v2.Summary)
	}
	if v2.Diff.Sections["background"].Change != "changed" {
		t.Errorf("v2 background change = %q, want changed", v2.Diff.Sections["background"].Change)
	}
	if _, listed := v2.Diff.Sections["goals"]; listed {
		t.Errorf("v2 diff lists the untouched goals section: %v", v2.Diff.Sections)
	}
	var counted int
	for _, n := range v2.Diff.Counts {
		counted += n
	}
	if counted != len(v2.Diff.Sections) {
		t.Errorf("v2 counts %v do not add up to its %d listed sections", v2.Diff.Counts, len(v2.Diff.Sections))
	}
}

// mustCommit posts a commit and requires 201.
func mustCommit(t *testing.T, ts *httptest.Server, token, prdPath, body string) {
	t.Helper()
	resp := doJSON(t, ts, token, http.MethodPost, prdPath+"/commit", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("commit %s: status %d, want 201; body %s", body, resp.StatusCode, b)
	}
}
