package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// FEAT-01 (review 3): the planner drafts in the language the coaching
// conversation chose. Build resolves it from the caller's own conversation
// on the PRD; expand walks node → project → PRD back to that conversation.
// A caller with no conversation (and a project no PRD stands behind) plans
// in zh, the planner's long-standing only language.

type roadmapLocaleFixture struct {
	st      *store.Store
	tokens  *auth.TokenIssuer
	token   string
	member  *store.User
	prdID   uuid.UUID
	nodeID  uuid.UUID
	memberT string
}

func newRoadmapLocaleFixture(t *testing.T, locale string) roadmapLocaleFixture {
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

	user, err := st.CreateUser(ctx, "rm-locale-"+uuid.NewString()+"@test.local", "x", "rm-locale")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, user.ID, "Roadmap locale WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	member, err := st.CreateUser(ctx, "rm-member-"+uuid.NewString()+"@test.local", "x", "rm-member")
	if err != nil {
		t.Fatalf("create member: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, member.ID, ws.Code); err != nil {
		t.Fatalf("join workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, user.ID, ws.ID, "Planned in a language", "content")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	prd, _, err := st.ConvertIdeaWithLocale(ctx, user.ID, idea.ID, locale)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}
	project, err := st.EnsureProjectForPrd(ctx, user.ID, prd.ID)
	if err != nil {
		t.Fatalf("ensure project: %v", err)
	}
	node, err := st.CreateRoadmapNode(ctx, user.ID, project.ID, nil, "Root", "")
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	tokens := auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour)
	token, err := tokens.MintAccessToken(user.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	memberT, err := tokens.MintAccessToken(member.ID)
	if err != nil {
		t.Fatalf("mint member token: %v", err)
	}
	return roadmapLocaleFixture{st: st, tokens: tokens, token: token, member: member,
		prdID: prd.ID, nodeID: node.ID, memberT: memberT}
}

// localePlanner records the locale each planner call received.
type localePlanner struct {
	plannedLocale  string
	expandedLocale string
}

func (p *localePlanner) PlanMVP(_ context.Context, locale, _ string, _ map[string]string) (*store.RoadmapPlanNode, error) {
	p.plannedLocale = locale
	return &store.RoadmapPlanNode{Title: "root"}, nil
}

func (p *localePlanner) ExpandNode(_ context.Context, locale, _, _, _ string) ([]store.RoadmapPlanNode, error) {
	p.expandedLocale = locale
	return []store.RoadmapPlanNode{{Title: "child"}}, nil
}

func (f roadmapLocaleFixture) serve(t *testing.T, planner RoadmapPlanner) *httptest.Server {
	t.Helper()
	srv := NewServer(f.st, f.tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, planner)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func (f roadmapLocaleFixture) post(t *testing.T, ts *httptest.Server, token, path string) int {
	t.Helper()
	resp := doJSON(t, ts, token, http.MethodPost, path, "")
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	return resp.StatusCode
}

func TestBuildAndExpand_FollowTheConversationsLocale(t *testing.T) {
	for _, tc := range []struct {
		locale string
	}{
		{store.LocaleEN},
		{store.LocaleZH},
	} {
		f := newRoadmapLocaleFixture(t, tc.locale)
		planner := &localePlanner{}
		ts := f.serve(t, planner)

		if status := f.post(t, ts, f.token, "/api/v1/prds/"+f.prdID.String()+"/build"); status != http.StatusOK {
			t.Fatalf("%s build: status %d", tc.locale, status)
		}
		if planner.plannedLocale != tc.locale {
			t.Fatalf("%s build planned in %q, want the conversation's locale", tc.locale, planner.plannedLocale)
		}

		if status := f.post(t, ts, f.token, "/api/v1/roadmap/"+f.nodeID.String()+"/expand"); status != http.StatusOK {
			t.Fatalf("%s expand: status %d", tc.locale, status)
		}
		if planner.expandedLocale != tc.locale {
			t.Fatalf("%s expand ran in %q, want the PRD's conversation locale", tc.locale, planner.expandedLocale)
		}
	}
}

// The member never coached this PRD, so they have no conversation to inherit
// a locale from: the planner stays zh for them.
func TestBuild_NoConversationPlansInChinese(t *testing.T) {
	f := newRoadmapLocaleFixture(t, store.LocaleEN)
	planner := &localePlanner{}
	ts := f.serve(t, planner)

	if status := f.post(t, ts, f.memberT, "/api/v1/prds/"+f.prdID.String()+"/build"); status != http.StatusOK {
		t.Fatalf("member build: status %d", status)
	}
	if planner.plannedLocale != store.LocaleZH {
		t.Fatalf("a builder with no conversation planned in %q, want zh", planner.plannedLocale)
	}
}
