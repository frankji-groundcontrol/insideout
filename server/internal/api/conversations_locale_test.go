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

// FEAT-01 (review 3): PRODUCT.md makes English and Simplified Chinese
// first-class product languages, but the conversation carried no locale at
// all, so the coach had nothing to select its prompt language from. These
// tests pin the storage half of the thread: the locale sent at creation is
// the locale the conversation keeps, canonicalized to the stored two-value
// form, while a create with no locale (every pre-locale client, and both
// conversation-creating routes) stays zh — the only language the coach has
// spoken so far.

type localeThreadFixture struct {
	st     *store.Store
	ts     *httptest.Server
	ws     *store.Workspace
	prd    *store.Prd
	author *store.User
	admin  *store.User
	tokens *auth.TokenIssuer
}

func newLocaleThreadFixture(t *testing.T) localeThreadFixture {
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
	f := localeThreadFixture{st: st, author: mk("author"), admin: mk("localeadmin")}
	f.ws, err = st.CreateWorkspace(ctx, f.author.ID, "Locale thread", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, f.admin.ID, f.ws.Code); err != nil {
		t.Fatalf("join workspace: %v", err)
	}
	if err := st.UpdateMemberRole(ctx, f.author.ID, f.ws.ID, f.admin.ID, "admin"); err != nil {
		t.Fatalf("promote admin: %v", err)
	}
	idea, err := st.CreateIdea(ctx, f.author.ID, f.ws.ID, "Locale thread idea", "a body")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	if f.prd, _, err = st.ConvertIdea(ctx, f.author.ID, idea.ID); err != nil {
		t.Fatalf("convert idea: %v", err)
	}

	f.tokens = auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour)
	srv := NewServer(st, f.tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, agent.NewTemplateRoadmapPlanner())
	f.ts = httptest.NewServer(srv.Handler())
	t.Cleanup(f.ts.Close)
	return f
}

// send performs one request as u and returns the status plus the raw body.
func (f localeThreadFixture) send(t *testing.T, u *store.User, method, path, body string) (int, []byte) {
	t.Helper()
	token, err := f.tokens.MintAccessToken(u.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	resp := doJSON(t, f.ts, token, method, path, body)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s body: %v", path, err)
	}
	return resp.StatusCode, raw
}

// conversationLocale GETs one conversation view and returns its locale.
func (f localeThreadFixture) conversationLocale(t *testing.T, u *store.User, path string) string {
	t.Helper()
	status, raw := f.send(t, u, http.MethodGet, path, "")
	if status != http.StatusOK {
		t.Fatalf("GET %s: status %d, want 200", path, status)
	}
	var view struct {
		Locale string `json:"locale"`
	}
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return view.Locale
}

func TestStartPrdConversation_StoresTheRequestedLocale(t *testing.T) {
	f := newLocaleThreadFixture(t)

	// The client sends the app locale tag it already runs in ('en-US');
	// the conversation keeps the canonical stored form.
	status, raw := f.send(t, f.admin, http.MethodPost, "/api/v1/prds/"+f.prd.ID.String()+"/conversation", `{"locale":"en-US"}`)
	if status != http.StatusCreated {
		t.Fatalf("admin POST with locale: status %d, want 201", status)
	}
	var view struct {
		Locale string `json:"locale"`
	}
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatalf("decode started conversation: %v", err)
	}
	if view.Locale != "en" {
		t.Fatalf("started conversation locale = %q, want the canonical \"en\"", view.Locale)
	}
}

func TestStartPrdConversation_WithoutLocaleStaysChinese(t *testing.T) {
	f := newLocaleThreadFixture(t)

	// Every pre-locale client posts no body at all; the zh default keeps
	// those conversations coaching in the language they always used.
	status, raw := f.send(t, f.admin, http.MethodPost, "/api/v1/prds/"+f.prd.ID.String()+"/conversation", "")
	if status != http.StatusCreated {
		t.Fatalf("bodyless admin POST: status %d, want 201", status)
	}
	var view struct {
		Locale string `json:"locale"`
	}
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatalf("decode started conversation: %v", err)
	}
	if view.Locale != "zh" {
		t.Fatalf("bodyless create locale = %q, want the zh default", view.Locale)
	}
}

func TestConvertedConversation_KeepsTheLocaleSentAtConversion(t *testing.T) {
	f := newLocaleThreadFixture(t)

	idea, err := f.st.CreateIdea(context.Background(), f.author.ID, f.ws.ID, "An English one", "body")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	status, raw := f.send(t, f.author, http.MethodPost, "/api/v1/ideas/"+idea.ID.String()+"/convert", `{"locale":"en-US"}`)
	if status != http.StatusCreated {
		t.Fatalf("convert with locale: status %d, want 201", status)
	}
	var converted struct {
		ConversationID string `json:"conversationId"`
	}
	if err := json.Unmarshal(raw, &converted); err != nil || converted.ConversationID == "" {
		t.Fatalf("decode converted conversation id from %s: %v", raw, err)
	}
	if got := f.conversationLocale(t, f.author, "/api/v1/conversations/"+converted.ConversationID); got != "en" {
		t.Fatalf("converted conversation locale = %q, want \"en\"", got)
	}

	// The fixture's own idea was converted with no locale at all: its
	// conversation stays zh (back-compat with every existing one).
	if got := f.conversationLocale(t, f.author, "/api/v1/prds/"+f.prd.ID.String()+"/conversation"); got != "zh" {
		t.Fatalf("no-locale converted conversation = %q, want the zh default", got)
	}
}
