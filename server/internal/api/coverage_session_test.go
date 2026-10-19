package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
)

// send is post with the cookies a browser would attach, and the whole response
// kept so its Set-Cookie headers can be read.
func (w *refreshWorld) send(method, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		r.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	w.h.ServeHTTP(rec, r)
	return rec
}

func setCookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// expectCleared fails the test unless the response told the browser to drop
// both auth cookies.
func expectCleared(t *testing.T, what string, rec *httptest.ResponseRecorder) {
	t.Helper()
	for _, name := range []string{accessCookieName, refreshCookieName} {
		if c := setCookie(rec, name); c == nil || c.Value != "" || c.MaxAge >= 0 {
			t.Fatalf("%s: cookie %s = %+v, want it cleared", what, name, c)
		}
	}
}

// Logout revokes the session its refresh token belongs to, and only that one:
// another login of the same user keeps working. It is idempotent: a call with
// no usable token still answers 200 and still clears the browser's cookies.
func TestLogoutRevokesOnlyTheSessionItIsHanded(t *testing.T) {
	w := newRefreshWorld(t, io.Discard)
	testlock.RequireLocalHost(t, w.st.Pool)
	u := w.newUser(t)
	byBody, other, byCookie := w.login(t, u), w.login(t, u), w.login(t, u)
	logout := func(body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		rec := w.send(http.MethodPost, "/api/v1/auth/logout", body, cookies...)
		if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"ok":true}` {
			t.Fatalf("logout answered %d %s, want 200 {\"ok\":true}", rec.Code, rec.Body.String())
		}
		expectCleared(t, "logout", rec)
		return rec
	}

	logout(`{"refreshToken":"` + byBody + `"}`)
	if status, _, _ := w.refresh(byBody); status != http.StatusUnauthorized {
		t.Fatalf("a token that was logged out still refreshes (status %d)", status)
	}
	other = w.rotate(t, other) // another login of the same user is untouched

	// The cookie is the credential when both are sent, as it is for a refresh.
	logout(`{"refreshToken":"`+other+`"}`, &http.Cookie{Name: refreshCookieName, Value: byCookie})
	if status, _, _ := w.refresh(byCookie); status != http.StatusUnauthorized {
		t.Fatalf("the cookie's session still refreshes (status %d) after logging out by cookie", status)
	}

	for name, body := range map[string]string{
		"no body":       "",
		"not json":      "refresh=please",
		"an unknown id": `{"refreshToken":"never-issued"}`,
		"blank token":   `{"refreshToken":"  "}`,
		"unknown field": `{"refreshToken":"` + other + `","all":true}`,
	} {
		t.Run(name, func(t *testing.T) { logout(body) })
	}
	w.rotate(t, other) // none of those touched a live session
}

func TestRefreshTakesItsTokenFromTheCookieOrTheBody(t *testing.T) {
	w := newRefreshWorld(t, io.Discard)
	testlock.RequireLocalHost(t, w.st.Pool)
	u := w.newUser(t)
	refresh := func(body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		return w.send(http.MethodPost, "/api/v1/auth/refresh", body, cookies...)
	}

	for name, body := range map[string]string{
		"no body":       "",
		"empty object":  `{}`,
		"blank token":   `{"refreshToken":"   "}`,
		"truncated":     `{"refreshToken":`,
		"wrong type":    `{"refreshToken":5}`,
		"unknown field": `{"token":"x"}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := refresh(body)
			if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "no refresh token") {
				t.Fatalf("refresh with %s = %d %s, want 401 no refresh token", name, rec.Code, rec.Body.String())
			}
		})
	}

	rec := refresh(`{"refreshToken":"never-issued"}`)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "expired or already used") {
		t.Fatalf("refresh with an unknown token = %d %s, want 401 expired or already used", rec.Code, rec.Body.String())
	}
	expectCleared(t, "refresh with an unknown token", rec)

	// A browser sends only the cookie; the pair that comes back is the new one,
	// in cookies a script cannot read.
	first := w.login(t, u)
	rec = refresh("", &http.Cookie{Name: refreshCookieName, Value: first})
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh by cookie = %d %s, want 200", rec.Code, rec.Body.String())
	}
	next := decodeAs[struct{ AccessToken, RefreshToken string }](t, rec.Body.String())
	rc, ac := setCookie(rec, refreshCookieName), setCookie(rec, accessCookieName)
	if rc == nil || rc.Value != next.RefreshToken || !rc.HttpOnly || rc.Path != "/api/v1/auth" ||
		ac == nil || ac.Value != next.AccessToken || !ac.HttpOnly || ac.Path != "/" || next.RefreshToken == first {
		t.Fatalf("cookies after a refresh: refresh %+v, access %+v, body %+v: want the rotated pair, HttpOnly", rc, ac, next)
	}
	if status, body := w.me(next.AccessToken); status != http.StatusOK || !strings.Contains(body, u.ID.String()) {
		t.Fatalf("the access token that came back: %d %s, want it to authenticate its user", status, body)
	}

	// The spent token is refused, and its refusal does not cost the winner's pair.
	rec = refresh(`{"refreshToken":"` + first + `"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("a spent token refreshed again: %d %s", rec.Code, rec.Body.String())
	}
	// Inside the reuse grace the 401 sets and deletes nothing: the cookies in the
	// browser are the winner's now (rv4-FR-F100).
	for _, name := range []string{accessCookieName, refreshCookieName} {
		if c := setCookie(rec, name); c != nil {
			t.Fatalf("a spent token inside the grace: Set-Cookie %s = %+v, want the winner's cookies left alone", name, c)
		}
	}

	// The cookie wins when the body carries something else.
	rec = refresh(`{"refreshToken":"garbage"}`, &http.Cookie{Name: refreshCookieName, Value: next.RefreshToken})
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh with a good cookie and a bad body = %d %s, want the cookie to win", rec.Code, rec.Body.String())
	}
}

// me calls a protected route with whatever credentials the caller attaches.
func (w *refreshWorld) me(bearer string, cookies ...*http.Cookie) (int, string) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	if bearer != "" {
		r.Header.Set("Authorization", "Bearer "+bearer)
	}
	for _, c := range cookies {
		r.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	w.h.ServeHTTP(rec, r)
	return rec.Code, rec.Body.String()
}

// A protected route takes a bearer token (the CLI, the apps) or the access
// cookie (the browser), and refuses everything else with a message that says
// whether credentials were missing or merely no good.
func TestProtectedRoutesNeedAValidAccessToken(t *testing.T) {
	w := newRefreshWorld(t, io.Discard)
	testlock.RequireLocalHost(t, w.st.Pool)
	u := w.newUser(t)
	mint := func(secret string, ttl time.Duration) string {
		t.Helper()
		token, err := auth.NewTokenIssuer(secret, ttl).MintAccessToken(u.ID)
		if err != nil {
			t.Fatalf("mint access token: %v", err)
		}
		return token
	}
	good := mint(routeTestSecret, time.Hour)

	if status, body := w.me(good); status != http.StatusOK || !strings.Contains(body, u.ID.String()) {
		t.Fatalf("bearer token: %d %s, want 200 with the user", status, body)
	}
	if status, body := w.me("", &http.Cookie{Name: accessCookieName, Value: good}); status != http.StatusOK || !strings.Contains(body, u.ID.String()) {
		t.Fatalf("access cookie: %d %s, want 200 with the user", status, body)
	}

	for name, c := range map[string]struct {
		bearer string
		cookie *http.Cookie
		want   string
	}{
		"nothing at all":       {want: "authentication required"},
		"an empty cookie":      {cookie: &http.Cookie{Name: accessCookieName}, want: "authentication required"},
		"not a token":          {bearer: "not-a-token", want: "invalid or expired session"},
		"an expired token":     {bearer: mint(routeTestSecret, -time.Minute), want: "invalid or expired session"},
		"another server's key": {bearer: mint("another-32-byte-or-longer-secret!", time.Hour), want: "invalid or expired session"},
		"a bad cookie":         {cookie: &http.Cookie{Name: accessCookieName, Value: "not-a-token"}, want: "invalid or expired session"},
		"a refresh token":      {bearer: w.login(t, u), want: "invalid or expired session"},
		"the bearer wins":      {bearer: "not-a-token", cookie: &http.Cookie{Name: accessCookieName, Value: good}, want: "invalid or expired session"},
	} {
		t.Run(name, func(t *testing.T) {
			var cookies []*http.Cookie
			if c.cookie != nil {
				cookies = append(cookies, c.cookie)
			}
			status, body := w.me(c.bearer, cookies...)
			if status != http.StatusUnauthorized || !strings.Contains(body, c.want) {
				t.Fatalf("%s: %d %s, want 401 %q", name, status, body, c.want)
			}
		})
	}
}

func TestHealthzAnswersWithoutCredentialsAndFailsWithTheDatabase(t *testing.T) {
	w := newRefreshWorld(t, io.Discard)
	testlock.RequireLocalHost(t, w.st.Pool)
	rec := w.send(http.MethodGet, "/healthz", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("healthz = %d %s, want 200 {\"status\":\"ok\"}", rec.Code, rec.Body.String())
	}

	// The same route over a pool that has been closed: the probe must say so.
	st, err := store.Open(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	st.Close()
	down := NewServer(st, auth.NewTokenIssuer(routeTestSecret, time.Hour), &config.Config{PublicURL: "http://app.test"},
		slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil).Handler()
	rec = httptest.NewRecorder()
	down.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "database unreachable") {
		t.Fatalf("healthz with the database gone = %d %s, want 503 database unreachable", rec.Code, rec.Body.String())
	}
}
