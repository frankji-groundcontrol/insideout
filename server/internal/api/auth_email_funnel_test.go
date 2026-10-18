package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/google/uuid"
)

// TEST-02: request→consume round trips for the email entrance flows. The
// dev-mode mail seam (cfg.MailDevBody) logs every sent body as a JSON
// record, which stands in for the hosted provider: each test requests a
// link, reads the token out of the logged mail, and posts it back the way
// the emailed link would. Real store, real PostgreSQL — same gates as the
// other route tests (newAdminWorld also supplies the promoted admin the
// invitation flow needs).

// mailWorld is a store-backed Server whose dev-mode mail sender logs into
// a JSON buffer the tests can read; world keeps the admin route helpers.
type mailWorld struct {
	*Server
	world *routeWorld
	admin routeUser
	buf   *bytes.Buffer
}

func newMailWorld(t *testing.T) *mailWorld {
	t.Helper()
	w, admin := newAdminWorld(t)
	buf := &bytes.Buffer{}
	srv := NewServer(w.st, auth.NewTokenIssuer(routeTestSecret, time.Hour),
		&config.Config{
			PublicURL:   "http://app.test",
			MailDevBody: true,
			AccessTTL:   15 * time.Minute,
			RefreshTTL:  720 * time.Hour,
		},
		slog.New(slog.NewJSONHandler(buf, nil)), nil, nil)
	return &mailWorld{Server: srv, world: w, admin: admin, buf: buf}
}

// sentMail is one message the dev-mode sender logged.
type sentMail struct {
	To      string
	Subject string
	Text    string
}

// lastMail decodes the most recent dev-mode mail record — the closest a
// test can get to what a hosted provider would receive.
func (m *mailWorld) lastMail(t *testing.T) sentMail {
	t.Helper()
	m.mail.Wait() // the emailed requests answer first and send from the mail queue (rv4-REL-07)
	var last sentMail
	for _, line := range strings.Split(m.buf.String(), "\n") {
		if !strings.Contains(line, `"mail dev mode`) {
			continue
		}
		var rec struct {
			To      string `json:"to"`
			Subject string `json:"subject"`
			Text    string `json:"text"`
		}
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("decode dev-mode mail record %q: %v", line, err)
		}
		last = sentMail{To: rec.To, Subject: rec.Subject, Text: rec.Text}
	}
	if last.Subject == "" {
		t.Fatal("no dev-mode mail was logged — the send never reached the mail sender")
	}
	return last
}

// tokenFrom returns the one-time token on the mailed link line that
// starts with prefix (the link is always a line of its own).
func tokenFrom(t *testing.T, mail sentMail, prefix string) string {
	t.Helper()
	for _, line := range strings.Split(mail.Text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return strings.TrimPrefix(strings.TrimSpace(line), prefix)
		}
	}
	t.Fatalf("no link starting with %q in mailed text:\n%s", prefix, mail.Text)
	return ""
}

// assertSessionCookies fails unless the response carries both auth
// cookies, non-empty — the visible half of issueSession.
func assertSessionCookies(t *testing.T, what string, rec *httptest.ResponseRecorder) {
	t.Helper()
	values := map[string]string{}
	for _, c := range rec.Result().Cookies() {
		values[c.Name] = c.Value
	}
	for _, name := range []string{accessCookieName, refreshCookieName} {
		if values[name] == "" {
			t.Errorf("%s: session cookie %s missing or empty; cookies %v", what, name, values)
		}
	}
}

// postJSONBearer posts a JSON body with a bearer access token.
func postJSONBearer(s *Server, token, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	return w
}

// TestMagicLinkRoundTrip requests a sign-in link, consumes the mailed
// token, and asserts the session cookies — then that the spent token is
// dead.
func TestMagicLinkRoundTrip(t *testing.T) {
	m := newMailWorld(t)
	email := "roundtrip-magic-" + uuid.NewString() + "@test.local"
	if _, err := m.store.CreateUser(context.Background(), email, "x", "magic-roundtrip"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	if w := postJSON(m.Server, "/api/v1/auth/magic-link", `{"email":"`+email+`"}`); w.Code != http.StatusOK {
		t.Fatalf("magic-link request: status %d, body %s", w.Code, w.Body.String())
	}
	token := tokenFrom(t, m.lastMail(t), "http://app.test/magic-link?token=")

	w := postJSON(m.Server, "/api/v1/auth/magic-link/consume", `{"token":"`+token+`"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("magic-link consume: status %d, body %s", w.Code, w.Body.String())
	}
	assertSessionCookies(t, "magic-link consume", w)

	// The link works once: a replay of the same token must be refused.
	if w := postJSON(m.Server, "/api/v1/auth/magic-link/consume", `{"token":"`+token+`"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("magic-link replay: status %d, want 401", w.Code)
	}
}

// TestPasswordResetRoundTrip requests a reset link, resets the password
// with the mailed token, and proves the change by logging in: the old
// password fails, the new one opens a session.
func TestPasswordResetRoundTrip(t *testing.T) {
	m := newMailWorld(t)
	email := "roundtrip-reset-" + uuid.NewString() + "@test.local"
	if _, err := m.store.CreateUser(context.Background(), email, "x", "reset-roundtrip"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	if w := postJSON(m.Server, "/api/v1/auth/forgot-password", `{"email":"`+email+`"}`); w.Code != http.StatusOK {
		t.Fatalf("forgot-password request: status %d, body %s", w.Code, w.Body.String())
	}
	token := tokenFrom(t, m.lastMail(t), "http://app.test/reset-password?token=")

	w := postJSON(m.Server, "/api/v1/auth/reset-password", `{"token":"`+token+`","password":"new-password-1"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("reset-password: status %d, body %s", w.Code, w.Body.String())
	}
	if w := postJSON(m.Server, "/api/v1/auth/login", `{"email":"`+email+`","password":"old-password-1"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("login with the old password: status %d, want 401", w.Code)
	}
	w = postJSON(m.Server, "/api/v1/auth/login", `{"email":"`+email+`","password":"new-password-1"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("login with the new password: status %d, body %s", w.Code, w.Body.String())
	}
	assertSessionCookies(t, "login after reset", w)
}

// TestInvitationRoundTrip mints an invitation as the admin, registers
// with the code out of the mailed link, and asserts the session plus the
// invitation being spent in the admin's list.
func TestInvitationRoundTrip(t *testing.T) {
	m := newMailWorld(t)
	email := "roundtrip-invitee-" + uuid.NewString() + "@test.local"

	// register only consumes the invite in invitation_only mode; the row
	// is shared global state, so save and restore it like the store suite.
	status, body := m.world.call(m.admin, http.MethodGet, "/api/v1/admin/settings", "")
	expect(t, "get signup mode", status, body, http.StatusOK)
	prior := decodeAs[map[string]string](t, body)["signupMode"]
	if prior != "invitation_only" {
		if status, body := m.world.call(m.admin, http.MethodPut, "/api/v1/admin/settings", `{"signupMode":"invitation_only"}`); status != http.StatusOK {
			t.Fatalf("set invitation_only: status %d, body %s", status, body)
		}
		t.Cleanup(func() {
			m.world.call(m.admin, http.MethodPut, "/api/v1/admin/settings", `{"signupMode":"`+prior+`"}`)
		})
	}

	if w := postJSONBearer(m.Server, m.admin.token, "/api/v1/admin/invitations", `{"email":"`+email+`"}`); w.Code != http.StatusCreated {
		t.Fatalf("create invitation: status %d, body %s", w.Code, w.Body.String())
	}
	code := tokenFrom(t, m.lastMail(t), "http://app.test/register?invite=")

	w := postJSON(m.Server, "/api/v1/auth/register",
		`{"email":"`+email+`","password":"invitee-pass-1","username":"invitee_`+uuid.NewString()[:8]+`","inviteCode":"`+code+`"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("register with the invited code: status %d, body %s", w.Code, w.Body.String())
	}
	assertSessionCookies(t, "register", w)

	// The invitation is spent: the admin's list shows it used.
	status, body = m.world.call(m.admin, http.MethodGet, "/api/v1/admin/invitations", "")
	expect(t, "list invitations", status, body, http.StatusOK)
	used := false
	for _, inv := range decodeAs[[]struct {
		Email  string
		UsedAt *string
	}](t, body) {
		if inv.Email == email {
			used = inv.UsedAt != nil
		}
	}
	if !used {
		t.Fatalf("invitation for %s not marked used after registering: %s", email, body)
	}
}

var loginCodeInText = regexp.MustCompile(`code is (\d{6})`)

// TestLoginCodeRoundTrip requests a sign-in code, verifies the mailed
// digits, and asserts the session — then that the code is dead.
func TestLoginCodeRoundTrip(t *testing.T) {
	m := newMailWorld(t)
	email := "roundtrip-code-" + uuid.NewString() + "@test.local"
	if _, err := m.store.CreateUser(context.Background(), email, "x", "code-roundtrip"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	if w := postJSON(m.Server, "/api/v1/auth/login-code", `{"email":"`+email+`"}`); w.Code != http.StatusOK {
		t.Fatalf("login-code request: status %d, body %s", w.Code, w.Body.String())
	}
	match := loginCodeInText.FindStringSubmatch(m.lastMail(t).Text)
	if match == nil {
		t.Fatalf("no 6-digit code in mailed text:\n%s", m.lastMail(t).Text)
	}

	w := postJSON(m.Server, "/api/v1/auth/login-code/verify", `{"email":"`+email+`","code":"`+match[1]+`"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("login-code verify: status %d, body %s", w.Code, w.Body.String())
	}
	assertSessionCookies(t, "login-code verify", w)

	if w := postJSON(m.Server, "/api/v1/auth/login-code/verify", `{"email":"`+email+`","code":"`+match[1]+`"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("login-code replay: status %d, want 401", w.Code)
	}
}
