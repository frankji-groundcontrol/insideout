package apiclient

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/api"
	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// Review-4 rv4-FR-F70, the client half: Client.Logout.

type rvApiClientLogouter interface{ Logout() error }

// rv4-FR-F70 (client half): `insideout logout` only deleted the local file, so a
// copy of it (a backup, a stolen laptop) kept a refresh token that renews for
// weeks. Logout revokes the session behind the client's refresh token on the
// server. The real handlers over real PostgreSQL; DATABASE_URL-gated.
func TestRvApiClientLogoutRevokesTheServerSession(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	email, password := "logout-"+uuid.NewString()+"@example.org", "pw-for-tests-"+uuid.NewString()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := st.CreateUser(ctx, email, hash, "logout-user"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	srv := api.NewServer(st, auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour),
		&config.Config{AccessTTL: time.Hour, RefreshTTL: time.Hour}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	base := ts.URL + "/api/v1"
	signIn := func() *Client {
		c := New(base)
		if err := c.Login(email, password); err != nil || c.RefreshToken() == "" {
			t.Fatalf("login: %v (refresh token %q)", err, c.RefreshToken())
		}
		return c
	}
	renewStatus := func(refresh string) int {
		resp, err := http.Post(base+"/auth/refresh", "application/json", strings.NewReader(`{"refreshToken":"`+refresh+`"}`))
		if err != nil {
			t.Fatalf("refresh: %v", err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	leaving, staying := signIn(), signIn() // two sessions of one user: the one that logs out, and a control
	spent := leaving.RefreshToken()        // Logout empties the client; the server has to refuse this one afterwards

	if err := rvApiClientHas[rvApiClientLogouter](t, leaving, "Logout").Logout(); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	if got := renewStatus(spent); got != http.StatusUnauthorized {
		t.Errorf("after Logout the refresh token still renews: HTTP %d, want 401", got)
	}
	if got := renewStatus(staying.RefreshToken()); got != http.StatusOK {
		t.Errorf("Logout ended another session of the user: HTTP %d, want 200", got)
	}
}

// Logout is a client method like the rest: the escape sweep lists it too, with
// the refresh token as the hostile value (it travels in the body, never the URL).
func init() {
	sweep = append(sweep, sweepCase{
		name: "Logout", method: "POST", shape: "/auth/logout",
		call: func(c *Client, h string) error {
			own := *c // the sweep's client keeps its own refresh token
			own.refresh = h
			return own.Logout()
		},
	})
}

// What Logout sends and how it ends. The refresh token goes in the body, no
// bearer; no refresh token, no request. A nil return means the client holds no
// credentials, and a 401 (the handler never answers one: something in front of
// the server did) is the same "already signed out". Any failure leaves the
// tokens, so a retry presents the same refresh token instead of reporting a
// revocation that never happened. An unanswered request is told apart from a
// refusal, and a cancelled one from both (the CLI words each differently).
func TestRvApiClientLogoutRequestAndFailures(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.Path+" ["+r.Header.Get("Authorization")+"] "+string(body))
		code := status
		mu.Unlock()
		w.WriteHeader(code)
	}))
	t.Cleanup(srv.Close)
	setStatus := func(code int) { mu.Lock(); status = code; mu.Unlock() }
	signedIn := func(base string) *Client {
		c := New(base)
		c.SetToken("access")
		c.refresh = "ref-1"
		return c
	}
	requests := func() []string { mu.Lock(); defer mu.Unlock(); return append([]string(nil), seen...) }
	cleared := func(what string, c *Client) {
		t.Helper()
		if c.Token() != "" || c.RefreshToken() != "" {
			t.Errorf("%s: the client holds access %q and refresh %q, want neither", what, c.Token(), c.RefreshToken())
		}
	}
	kept := func(what string, c *Client) {
		t.Helper()
		if c.Token() != "access" || c.RefreshToken() != "ref-1" {
			t.Errorf("%s: the client holds access %q and refresh %q, want both kept for a retry", what, c.Token(), c.RefreshToken())
		}
	}

	ok := signedIn(srv.URL)
	if err := ok.Logout(); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if got, want := requests(), []string{`POST /auth/logout [] {"refreshToken":"ref-1"}`}; len(got) != 1 || got[0] != want[0] {
		t.Errorf("server saw %q, want %q", got, want)
	}
	cleared("answered 200", ok)

	anonymous := New(srv.URL)
	anonymous.SetToken("access") // an access token alone cannot be revoked
	if err := anonymous.Logout(); err != nil || len(requests()) != 1 {
		t.Errorf("without a refresh token: err = %v, %d requests, want nil and no request", err, len(requests())-1)
	}
	cleared("no refresh token", anonymous)

	setStatus(http.StatusUnauthorized)
	gated := signedIn(srv.URL)
	if err := gated.Logout(); err != nil {
		t.Errorf("answered 401: err = %v, want nil (already signed out)", err)
	}
	if n := len(requests()); n != 2 {
		t.Errorf("answered 401: %d requests so far, want one more than before and not a retry", n)
	}
	cleared("answered 401", gated)

	setStatus(http.StatusServiceUnavailable)
	refused := signedIn(srv.URL)
	if err := refused.Logout(); err == nil || !strings.Contains(err.Error(), "503") || errors.Is(err, ErrLogoutUnreachable) {
		t.Errorf("refused: err = %v, want the status named and not the unreachable sentinel", err)
	}
	kept("answered 503", refused)

	dead := httptest.NewServer(http.NotFoundHandler())
	dead.Close()
	silent := signedIn(dead.URL)
	if err := silent.Logout(); !errors.Is(err, ErrLogoutUnreachable) {
		t.Errorf("no answer: err = %v, want it to wrap ErrLogoutUnreachable", err)
	}
	kept("no answer", silent)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	gone := signedIn(srv.URL)
	gone.SetContext(ctx)
	if err := gone.Logout(); !errors.Is(err, context.Canceled) || errors.Is(err, ErrLogoutUnreachable) {
		t.Errorf("cancelled: err = %v, want context.Canceled and not the unreachable sentinel", err)
	}
	kept("cancelled", gone)
}
