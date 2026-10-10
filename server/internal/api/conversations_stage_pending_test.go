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

// The stage proposal and the parked finding resolutions must both be
// visible to and ratifiable by the person: the GET carries the parked
// pendingStage so a reload can show the proposal (USER-01), and confirming
// the stage is what applies the resolutions the model parked (AIDEV-07) —
// overrides become human-authored by construction.
type stageFixture struct {
	st     *store.Store
	ts     *httptest.Server
	prd    *store.Prd
	convID uuid.UUID
	owner  *store.User
	tokens *auth.TokenIssuer
}

func newStageFixture(t *testing.T) stageFixture {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)

	owner, err := st.CreateUser(ctx, "owner-"+uuid.NewString()+"@test.local", "x", "owner")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, owner.ID, "Stage confirm", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, owner.ID, ws.ID, "Ship a thing", "")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	prd, _, err := st.ConvertIdea(ctx, owner.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}

	// The real coach: the confirm path must route through it, not the bare
	// store, so the parked resolutions ride the same confirmation.
	tokens := auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(st, tokens, &config.Config{}, log, agent.New(st, nil, log), agent.NewTemplateRoadmapPlanner())
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	f := stageFixture{st: st, ts: ts, prd: prd, owner: owner, tokens: tokens}
	token, err := tokens.MintAccessToken(owner.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	resp := doJSON(t, ts, token, http.MethodGet, "/api/v1/prds/"+prd.ID.String()+"/conversation", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET conversation: status %d", resp.StatusCode)
	}
	var view struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
		t.Fatalf("decode conversation: %v", err)
	}
	f.convID = uuid.MustParse(view.ID)
	return f
}

func (f stageFixture) parkMeta(t *testing.T, meta string) {
	t.Helper()
	if err := f.st.UpdateConversationMeta(context.Background(), f.owner.ID, f.convID, json.RawMessage(meta)); err != nil {
		t.Fatalf("park meta: %v", err)
	}
}

func (f stageFixture) confirm(t *testing.T, stage string) int {
	t.Helper()
	token, err := f.tokens.MintAccessToken(f.owner.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	resp := doJSON(t, f.ts, token, http.MethodPost, "/api/v1/conversations/"+f.convID.String()+"/stage", `{"stage":"`+stage+`"}`)
	defer resp.Body.Close()
	return resp.StatusCode
}

func (f stageFixture) meta(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	conv, err := f.st.GetConversationForOwner(context.Background(), f.convID, f.owner.ID)
	if err != nil {
		t.Fatalf("reload conversation: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(conv.Meta, &m); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	return m
}

func TestGetConversationExposesPendingStage(t *testing.T) {
	f := newStageFixture(t)
	f.parkMeta(t, `{"pendingStage":"draft"}`)

	token, err := f.tokens.MintAccessToken(f.owner.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	resp := doJSON(t, f.ts, token, http.MethodGet, "/api/v1/conversations/"+f.convID.String(), "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET: status %d", resp.StatusCode)
	}
	var view struct {
		Stage        string `json:"stage"`
		PendingStage string `json:"pendingStage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if view.Stage != "clarify" {
		t.Fatalf("stage = %q, want the unadvanced clarify", view.Stage)
	}
	if view.PendingStage != "draft" {
		t.Fatalf("pendingStage = %q, want the parked draft proposal", view.PendingStage)
	}
}

func TestConfirmStageAppliesParkedFindingResolutions(t *testing.T) {
	f := newStageFixture(t)
	f.parkMeta(t, `{
		"critic_findings": [
			{"id":"cf1","section":"background","severity":"blocking","kind":"defect","issue":"no problem stated","status":"open"},
			{"id":"cf2","section":"goals","severity":"major","kind":"omission","issue":"no goal","status":"open"}
		],
		"pendingFindingResolutions": [{"id":"cf1","status":"resolved"}],
		"pendingStage":"draft"
	}`)

	if status := f.confirm(t, "draft"); status != http.StatusOK {
		t.Fatalf("confirm: status %d", status)
	}

	meta := f.meta(t)
	var findings []map[string]any
	if err := json.Unmarshal(meta["critic_findings"], &findings); err != nil {
		t.Fatalf("unmarshal findings: %v", err)
	}
	byID := map[string]any{}
	for _, fnd := range findings {
		byID[fnd["id"].(string)] = fnd["status"]
	}
	if byID["cf1"] != "resolved" {
		t.Fatalf("cf1 status = %v, want resolved — the human's confirmation must apply what the model parked", byID["cf1"])
	}
	if byID["cf2"] != "open" {
		t.Fatalf("cf2 status = %v, want still open — only the parked resolution applies", byID["cf2"])
	}
	if _, parked := meta["pendingFindingResolutions"]; parked {
		t.Fatalf("pendingFindingResolutions still present after confirmation: %s", meta["pendingFindingResolutions"])
	}
	if _, stripped := meta["pendingStage"]; stripped {
		t.Fatalf("pendingStage still present after confirmation: %s", meta["pendingStage"])
	}
	var trail []map[string]any
	if raw, ok := meta["findingResolutionsApplied"]; ok {
		if err := json.Unmarshal(raw, &trail); err != nil {
			t.Fatalf("unmarshal trail: %v", err)
		}
	}
	if len(trail) != 1 || trail[0]["id"] != "cf1" {
		t.Fatalf("findingResolutionsApplied = %+v, want one cf1 entry recording the ratification", trail)
	}
}
