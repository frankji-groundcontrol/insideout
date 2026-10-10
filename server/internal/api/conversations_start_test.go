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

// A coach conversation is owner-only, and ConvertIdea makes exactly one: the
// converter's. GET /prds/{id}/conversation therefore 404s for every other
// teammate, and the panel had nothing to offer them. POST on the same path
// starts (or resumes) the caller's own conversation.
type startFixture struct {
	st        *store.Store
	ts        *httptest.Server
	prd       *store.Prd
	converter *store.User
	admin     *store.User
	member    *store.User
	stranger  *store.User
	tokens    *auth.TokenIssuer
}

func newStartFixture(t *testing.T) startFixture {
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

	mk := func(name string) *store.User {
		u, err := st.CreateUser(ctx, name+"-"+uuid.NewString()+"@test.local", "x", name)
		if err != nil {
			t.Fatalf("create user %s: %v", name, err)
		}
		return u
	}
	f := startFixture{st: st, converter: mk("converter"), admin: mk("admin"), member: mk("member"), stranger: mk("stranger")}

	ws, err := st.CreateWorkspace(ctx, f.converter.ID, "Coach start", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	for _, u := range []*store.User{f.admin, f.member} {
		if _, err := st.JoinWorkspace(ctx, u.ID, ws.Code); err != nil {
			t.Fatalf("join workspace: %v", err)
		}
	}
	if err := st.UpdateMemberRole(ctx, f.converter.ID, ws.ID, f.admin.ID, "admin"); err != nil {
		t.Fatalf("promote admin: %v", err)
	}
	idea, err := st.CreateIdea(ctx, f.converter.ID, ws.ID, "Ship a thing", "one path")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	if f.prd, _, err = st.ConvertIdea(ctx, f.converter.ID, idea.ID); err != nil {
		t.Fatalf("convert idea: %v", err)
	}

	f.tokens = auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour)
	srv := NewServer(st, f.tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, agent.NewTemplateRoadmapPlanner())
	f.ts = httptest.NewServer(srv.Handler())
	t.Cleanup(f.ts.Close)
	return f
}

// call sends one request as u and decodes the conversation it returns, when
// there is one.
func (f startFixture) call(t *testing.T, u *store.User, method string) (int, conversationView) {
	t.Helper()
	token, err := f.tokens.MintAccessToken(u.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	resp := doJSON(t, f.ts, token, method, "/api/v1/prds/"+f.prd.ID.String()+"/conversation", "")
	defer resp.Body.Close()
	var view conversationView
	if resp.StatusCode < 300 {
		if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
			t.Fatalf("%s decode: %v", method, err)
		}
	}
	return resp.StatusCode, view
}

func TestStartPrdConversation_AnotherAdminStartsOnceAndThenResumes(t *testing.T) {
	f := newStartFixture(t)

	if status, _ := f.call(t, f.admin, http.MethodGet); status != http.StatusNotFound {
		t.Fatalf("admin GET before starting: status %d, want 404", status)
	}

	status, first := f.call(t, f.admin, http.MethodPost)
	if status != http.StatusCreated {
		t.Fatalf("admin POST: status %d, want 201 for a new conversation", status)
	}
	if first.ID == "" || first.PrdID != f.prd.ID.String() || first.Stage != "clarify" || first.Status != "active" {
		t.Fatalf("started conversation = %+v, want an active clarify one on this PRD", first)
	}

	status, again := f.call(t, f.admin, http.MethodPost)
	if status != http.StatusOK || again.ID != first.ID {
		t.Fatalf("admin POST again: status %d id %s, want 200 and the same %s", status, again.ID, first.ID)
	}

	// The existing GET now resumes it, which is what the panel does on revisit.
	status, got := f.call(t, f.admin, http.MethodGet)
	if status != http.StatusOK || got.ID != first.ID {
		t.Fatalf("admin GET after starting: status %d id %s, want 200 and %s", status, got.ID, first.ID)
	}
}

func TestStartPrdConversation_ConverterResumesTheConvertedOne(t *testing.T) {
	f := newStartFixture(t)
	_, existing := f.call(t, f.converter, http.MethodGet)
	if existing.ID == "" {
		t.Fatal("converter has no conversation from ConvertIdea")
	}

	status, got := f.call(t, f.converter, http.MethodPost)
	if status != http.StatusOK || got.ID != existing.ID {
		t.Fatalf("converter POST: status %d id %s, want 200 and their own %s", status, got.ID, existing.ID)
	}
}

func TestStartPrdConversation_MemberIs403AndStrangerIs404(t *testing.T) {
	f := newStartFixture(t)

	if status, _ := f.call(t, f.member, http.MethodPost); status != http.StatusForbidden {
		t.Fatalf("plain member POST: status %d, want 403", status)
	}
	// Refused means refused: nothing was created behind the 403.
	if status, _ := f.call(t, f.member, http.MethodGet); status != http.StatusNotFound {
		t.Fatalf("plain member GET after a refused POST: status %d, want 404", status)
	}
	if status, _ := f.call(t, f.stranger, http.MethodPost); status != http.StatusNotFound {
		t.Fatalf("stranger POST: status %d, want 404 (existence not disclosed)", status)
	}
}
