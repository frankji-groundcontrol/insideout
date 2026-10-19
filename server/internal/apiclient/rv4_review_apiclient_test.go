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
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/api"
	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// Review-4 (multirole, apiclient cluster). The findings are pinned where they
// live: rv4-FR-F151/F164 and rv4-FR-F169/F165/F157 in refresh_test.go; the
// rest here.

// rvApiClientHas fetches a method the client has to offer as an interface, so a
// test that names a method the client does not have yet fails when it runs, not
// when the package compiles (the rest of the package keeps testing).
func rvApiClientHas[T any](t *testing.T, c *Client, what string) T {
	t.Helper()
	v, ok := any(c).(T)
	if !ok {
		t.Fatalf("*Client has no %s", what)
	}
	return v
}

type rvApiClientHinter interface{ SetUnauthorizedHint(string) }

// escape_test.go keeps the list of every exported Client method; a method added
// in review 4 registers itself here instead of in that file.
func init() {
	sweep = append(sweep, sweepCase{
		name: "SetUnauthorizedHint", method: "GET", shape: "/me",
		call: func(c *Client, h string) error {
			c.SetUnauthorizedHint(h) // whatever its text, a hint never reaches the request line
			defer c.SetUnauthorizedHint("")
			return drop(c.Whoami())
		},
	})
}

const rvApiClientHint = "your sign-in ended: call the auth_login tool again"

// rv4-FR-F152: the final 401 tells the user to run `insideout login`, which
// cannot cure a client whose credential is not the saved login (an MCP
// in-process sign-in). Such a caller replaces the advice, for its own client
// only.
func TestRvApiClientUnauthorizedHintIsPerClient(t *testing.T) {
	base, _ := sessionAPI(t, func(w http.ResponseWriter, _ string) { w.WriteHeader(http.StatusUnauthorized) })
	for name, token := range map[string]string{"signed in": "old", "signed out": ""} {
		t.Run(name, func(t *testing.T) {
			c := New(base)
			c.SetToken(token)
			rvApiClientHas[rvApiClientHinter](t, c, "SetUnauthorizedHint").SetUnauthorizedHint(rvApiClientHint)
			other := New(base) // another client of the same process keeps the default advice
			other.SetToken(token)

			_, err := c.Whoami()
			if !errors.Is(err, ErrUnauthorized) || !strings.Contains(err.Error(), rvApiClientHint) || strings.Contains(err.Error(), "insideout login") {
				t.Errorf("err = %v, want ErrUnauthorized carrying the hint and not the advice to run insideout login", err)
			}
			if _, err := other.Whoami(); err == nil || !strings.Contains(err.Error(), "insideout login") {
				t.Errorf("another client: err = %v, want the default advice", err)
			}
		})
	}
}

// The hint must outlive a renewal the server rejected (that disarms the client)
// and not depend on whether it was set before or after SetRenewal.
func TestRvApiClientHintSurvivesARejectedRenewal(t *testing.T) {
	for name, hintFirst := range map[string]bool{"hint before SetRenewal": true, "hint after SetRenewal": false} {
		t.Run(name, func(t *testing.T) {
			base, calls := sessionAPI(t, func(w http.ResponseWriter, _ string) { w.WriteHeader(http.StatusUnauthorized) })
			f := &file{access: "old", refresh: "ref-1"}
			c := New(base)
			c.SetToken(f.access)
			hinter := rvApiClientHas[rvApiClientHinter](t, c, "SetUnauthorizedHint")
			if hintFirst {
				hinter.SetUnauthorizedHint(rvApiClientHint)
			}
			c.SetRenewal(f.refresh, f.renewal())
			if !hintFirst {
				hinter.SetUnauthorizedHint(rvApiClientHint)
			}

			for i := 0; i < 2; i++ { // the first call spends the renewal, the second finds it gone
				_, err := c.Whoami()
				if !errors.Is(err, ErrUnauthorized) || !strings.Contains(err.Error(), rvApiClientHint) || strings.Contains(err.Error(), "insideout login") {
					t.Fatalf("call %d: err = %v, want ErrUnauthorized carrying the hint", i, err)
				}
			}
			if n := strings.Count(paths(calls()), "/auth/refresh"); n != 1 {
				t.Errorf("a rejected refresh token was offered %d times", n)
			}
		})
	}
}

// rv4-FR-F66: the server rotates the refresh token as it reads the renewal, so
// a call cancelled while the answer is on its way must still save the new pair:
// dropping the request leaves the stored login dead (the old token is spent and
// the new one never arrived).
func TestRvApiClientCancelledRenewalKeepsTheRotatedPair(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	var rotated atomic.Bool
	answered := make(chan struct{})
	base, _ := sessionAPI(t, func(w http.ResponseWriter, body string) {
		defer close(answered)
		cancel()                           // the caller gives up as the request lands...
		time.Sleep(300 * time.Millisecond) // ...and the answer takes a moment
		rotated.Store(strings.Contains(body, `"ref-1"`))
		swap(w, body)
	})
	f := &file{access: "old", refresh: "ref-1"}
	c := armed(base, f)
	c.SetContext(ctx)

	_, err := c.Whoami()

	select {
	case <-answered:
	case <-time.After(5 * time.Second):
		t.Fatal("the stub never answered the renewal")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want the call to end as cancelled", err)
	}
	if !rotated.Load() {
		t.Fatal("the stub did not rotate ref-1: the test set nothing up")
	}
	if f.access != "new" || f.refresh != "ref-2" || f.saved != 1 {
		t.Errorf("stored session = %+v, want the pair the server rotated to (new/ref-2) saved once", *f)
	}
}

// A caller that has gone before the renewal starts must not spend a refresh
// token on it: with the answer no longer waited for, the POST would otherwise be
// sent and rotate the stored login for nobody.
func TestRvApiClientCancelledBeforeTheRenewalSendsNothing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	base, calls := sessionAPI(t, swap)
	f := &file{access: "old", refresh: "ref-1"}
	r := f.renewal()
	reload := r.Reload
	r.Reload = func() (string, string, bool) { cancel(); return reload() } // the user gives up as the renewal begins
	c := New(base)
	c.SetToken(f.access)
	c.SetRenewal(f.refresh, r)
	c.SetContext(ctx)

	_, err := c.Whoami()

	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want the call to end as cancelled", err)
	}
	if strings.Contains(paths(calls()), "/auth/refresh") {
		t.Errorf("calls = %s: a refresh token was spent for a caller that had gone", paths(calls()))
	}
	if f.refresh != "ref-1" || f.saved != 0 {
		t.Errorf("stored session = %+v, want it untouched", *f)
	}
}

func TestRvApiClientWaitForRenewal(t *testing.T) {
	was := renewalDrain
	renewalDrain = 100 * time.Millisecond // the bound is twice this
	t.Cleanup(func() { renewalDrain = was })

	waited := func() time.Duration {
		start := time.Now()
		WaitForRenewal()
		return time.Since(start)
	}
	if d := waited(); d > 50*time.Millisecond {
		t.Errorf("with no renewal in flight it waited %v", d)
	}

	renewing.Lock() // a renewal in flight, finishing after 60 ms
	time.AfterFunc(60*time.Millisecond, renewing.Unlock)
	if d := waited(); d < 50*time.Millisecond || d > 190*time.Millisecond {
		t.Errorf("it waited %v for a renewal that ends after 60 ms", d)
	}

	renewing.Lock() // one that never ends: the wait is bounded, and does not take the lock
	defer renewing.Unlock()
	if d := waited(); d < 190*time.Millisecond || d > time.Second {
		t.Errorf("it waited %v for a renewal that never ends, want about 200 ms", d)
	}
}

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

	if err := rvApiClientHas[rvApiClientLogouter](t, leaving, "Logout").Logout(); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	if got := renewStatus(leaving.RefreshToken()); got != http.StatusUnauthorized {
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

// What Logout sends and how it fails: the refresh token and no bearer; no token,
// no request; an unanswered request is told apart from a refusal, and a
// cancelled one from both (the CLI words each differently).
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
	signedIn := func(base string) *Client {
		c := New(base)
		c.SetToken("access")
		c.refresh = "ref-1"
		return c
	}
	requests := func() []string { mu.Lock(); defer mu.Unlock(); return append([]string(nil), seen...) }

	if err := signedIn(srv.URL).Logout(); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if got, want := requests(), []string{`POST /auth/logout [] {"refreshToken":"ref-1"}`}; len(got) != 1 || got[0] != want[0] {
		t.Errorf("server saw %q, want %q", got, want)
	}

	anonymous := New(srv.URL)
	anonymous.SetToken("access") // an access token alone cannot be revoked
	if err := anonymous.Logout(); err != nil || len(requests()) != 1 {
		t.Errorf("without a refresh token: err = %v, %d requests, want nil and no request", err, len(requests())-1)
	}

	mu.Lock()
	status = http.StatusServiceUnavailable
	mu.Unlock()
	if err := signedIn(srv.URL).Logout(); err == nil || !strings.Contains(err.Error(), "503") || errors.Is(err, ErrLogoutUnreachable) {
		t.Errorf("refused: err = %v, want the status named and not the unreachable sentinel", err)
	}

	dead := httptest.NewServer(http.NotFoundHandler())
	dead.Close()
	if err := signedIn(dead.URL).Logout(); !errors.Is(err, ErrLogoutUnreachable) {
		t.Errorf("no answer: err = %v, want it to wrap ErrLogoutUnreachable", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	gone := signedIn(srv.URL)
	gone.SetContext(ctx)
	if err := gone.Logout(); !errors.Is(err, context.Canceled) || errors.Is(err, ErrLogoutUnreachable) {
		t.Errorf("cancelled: err = %v, want context.Canceled and not the unreachable sentinel", err)
	}
}
