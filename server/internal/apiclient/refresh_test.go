package apiclient

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// Review-4 USER-04, at the client: a 401 on an authenticated call is answered
// once by renewing the session (POST /auth/refresh, one-time token, rotated
// pair) and replaying the call. The black-box CLI tests and the real-server
// test live in cmd/insideout; these pin the edges.

type call struct{ method, path, auth, body string }

// swap answers /auth/refresh for the pair ref-1 with new/ref-2, once.
func swap(w http.ResponseWriter, body string) {
	if strings.Contains(body, `"ref-1"`) {
		w.Write([]byte(`{"ok":true,"accessToken":"new","refreshToken":"ref-2"}`))
		return
	}
	w.WriteHeader(http.StatusUnauthorized)
}

// sessionAPI accepts only "Bearer new" outside /auth and records every call.
func sessionAPI(t *testing.T, refresh func(w http.ResponseWriter, body string)) (string, func() []call) {
	t.Helper()
	var mu sync.Mutex
	var calls []call
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		calls = append(calls, call{r.Method, r.URL.Path, r.Header.Get("Authorization"), string(body)})
		mu.Unlock()
		switch {
		case r.URL.Path == "/auth/refresh":
			refresh(w, string(body))
		case r.URL.Path == "/auth/login":
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid credentials"}`))
		case r.Header.Get("Authorization") == "Bearer new" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusNoContent) // like POST /projects/{id}/cursor: no body to decode
		case r.Header.Get("Authorization") == "Bearer new":
			w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid or expired session"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL, func() []call {
		mu.Lock()
		defer mu.Unlock()
		return append([]call(nil), calls...)
	}
}

// file is the stored session the Renewal hooks stand for.
type file struct {
	access, refresh string
	saved           int
}

func (f *file) renewal() Renewal {
	return Renewal{
		Reload: func() (string, string, bool) { return f.access, f.refresh, f.refresh != "" },
		Save:   func(a, r string) { f.access, f.refresh = a, r; f.saved++ },
	}
}

func armed(base string, f *file) *Client {
	c := New(base)
	c.SetToken(f.access)
	c.SetRenewal(f.refresh, f.renewal())
	return c
}

func paths(calls []call) string {
	var out []string
	for _, c := range calls {
		out = append(out, c.method+" "+c.path+" "+c.auth)
	}
	return strings.Join(out, " | ")
}

func TestRenewalReplaysTheCallWithItsBodyAndTheNewToken(t *testing.T) {
	base, calls := sessionAPI(t, swap)
	f := &file{access: "old", refresh: "ref-1"}

	if err := armed(base, f).Cursor("p1", 1, 2); err != nil {
		t.Fatalf("Cursor: %v", err)
	}

	got := calls()
	if want := "POST /projects/p1/cursor Bearer old | POST /auth/refresh  | POST /projects/p1/cursor Bearer new"; paths(got) != want {
		t.Fatalf("calls = %s\nwant   %s", paths(got), want)
	}
	if got[0].body != `{"x":1,"y":2}` || got[2].body != got[0].body {
		t.Errorf("replayed body %q, original %q", got[2].body, got[0].body)
	}
	if f.access != "new" || f.refresh != "ref-2" || f.saved != 1 {
		t.Errorf("stored session = %+v, want the rotated pair saved once", *f)
	}
}

func TestRenewalHappensOncePerRefusal(t *testing.T) {
	base, calls := sessionAPI(t, func(w http.ResponseWriter, _ string) {
		w.Write([]byte(`{"ok":true,"accessToken":"still-bad","refreshToken":"ref-2"}`))
	})
	f := &file{access: "old", refresh: "ref-1"}

	_, err := armed(base, f).Whoami()

	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized after the one replay", err)
	}
	if got := paths(calls()); strings.Count(got, "/auth/refresh") != 1 || strings.Count(got, "GET /me") != 2 {
		t.Errorf("calls = %s, want one renewal and one replay", got)
	}
}

func TestRejectedRenewalIsFinalAndNotRepeated(t *testing.T) {
	base, calls := sessionAPI(t, func(w http.ResponseWriter, _ string) { w.WriteHeader(http.StatusUnauthorized) })
	f := &file{access: "old", refresh: "ref-1"}
	c := armed(base, f)

	for i := 0; i < 2; i++ {
		if _, err := c.Whoami(); !errors.Is(err, ErrUnauthorized) || !strings.Contains(err.Error(), "insideout login") {
			t.Fatalf("call %d: err = %v, want an ErrUnauthorized that says to log in", i, err)
		}
	}
	if n := strings.Count(paths(calls()), "/auth/refresh"); n != 1 {
		t.Errorf("a rejected refresh token was offered %d times", n)
	}
	if f.saved != 0 {
		t.Error("a rejected renewal must not write the stored session")
	}
}

func TestRenewalTroubleIsNotAVerdictOnTheSession(t *testing.T) {
	// 429 is the per-IP throttle on /auth/refresh, 503 a deploy in progress: neither
	// says anything about the session (rv4-FR-F151, rv4-FR-F164).
	for _, status := range []int{http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			base, calls := sessionAPI(t, func(w http.ResponseWriter, _ string) { w.WriteHeader(status) })
			f := &file{access: "old", refresh: "ref-1"}
			c := armed(base, f)

			for i := 0; i < 2; i++ {
				_, err := c.Whoami()
				if err == nil || errors.Is(err, ErrUnauthorized) || !strings.Contains(err.Error(), "/auth/refresh") ||
					!strings.Contains(err.Error(), strconv.Itoa(status)) || strings.Contains(err.Error(), "expired or was revoked") {
					t.Fatalf("call %d: err = %v, want the failed renewal named with its status, not a verdict on the session", i, err)
				}
			}

			if n := strings.Count(paths(calls()), "/auth/refresh"); n != 2 {
				t.Errorf("after a %d the client must keep its renewal (tried %d times, want 2)", status, n)
			}
			if f.saved != 0 || f.refresh != "ref-1" {
				t.Errorf("a refused renewal must leave the stored session alone: %+v", *f)
			}
		})
	}
}

func TestUnreachableRenewalIsReportedAsSuch(t *testing.T) {
	base, _ := sessionAPI(t, func(w http.ResponseWriter, _ string) {
		conn, _, _ := w.(http.Hijacker).Hijack()
		conn.Close()
	})
	_, err := armed(base, &file{access: "old", refresh: "ref-1"}).Whoami()
	if err == nil || !strings.Contains(err.Error(), "/auth/refresh") || errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want the failed renewal named, not a verdict on the session", err)
	}
}

func TestStoredPairIsAdoptedBeforeTheServerIsAsked(t *testing.T) {
	base, calls := sessionAPI(t, swap)
	f := &file{access: "new", refresh: "ref-9"} // another command renewed already
	c := New(base)
	c.SetToken("old")
	c.SetRenewal("ref-1", f.renewal())

	if _, err := c.Whoami(); err != nil {
		t.Fatalf("Whoami: %v", err)
	}
	if got := paths(calls()); got != "GET /me Bearer old | GET /me Bearer new" {
		t.Errorf("calls = %s, want the stored pair adopted with no /auth/refresh", got)
	}
}

func TestCallsWithoutABearerOrToAuthRoutesAreNotRenewed(t *testing.T) {
	base, calls := sessionAPI(t, swap)
	f := &file{refresh: "ref-1"} // no access token: the request carries no bearer
	c := armed(base, f)
	if _, err := c.Whoami(); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("anonymous Whoami: %v", err)
	}

	c.SetToken("old") // a bearer, but on /auth/login a 401 is the password being wrong
	err := c.Login("a@example.org", "wrong-password")

	if err == nil || !strings.Contains(err.Error(), "invalid credentials") || strings.Contains(err.Error(), "expired") {
		t.Errorf("Login err = %v, want the server's own words", err)
	}
	if strings.Contains(paths(calls()), "/auth/refresh") {
		t.Errorf("calls = %s: a call without a bearer, or to /auth/*, was renewed", paths(calls()))
	}
}

func TestUnarmedClientsAndOtherRefusalsAreNeverRenewed(t *testing.T) {
	var refreshes int
	status := http.StatusUnauthorized
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/refresh" {
			refreshes++
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)

	unarmed := New(srv.URL) // e.g. a client that merely logged in, or INSIDEOUT_TOKEN
	unarmed.SetToken("old")
	_, _ = unarmed.Whoami()

	status = http.StatusForbidden
	armedC := New(srv.URL)
	armedC.SetToken("old")
	armedC.SetRenewal("ref-1", (&file{}).renewal())
	_, err := armedC.Whoami()

	if refreshes != 0 {
		t.Errorf("/auth/refresh was called %d times", refreshes)
	}
	if err == nil || errors.Is(err, ErrUnauthorized) {
		t.Errorf("a 403 is not a 401: %v", err)
	}
}

func TestUnauthorizedNamesTheFix(t *testing.T) {
	base, _ := sessionAPI(t, swap)
	for name, token := range map[string]string{"signed in": "old", "signed out": ""} {
		c := New(base)
		c.SetToken(token)
		_, getErr := c.Whoami()
		_, rawErr := c.PrdExport("p1", "md", "")
		_, guideErr := c.Guide("p1") // rv4-FR-F169: it had a request path of its own, with a bare 401
		for _, err := range []error{getErr, rawErr, guideErr} {
			if !errors.Is(err, ErrUnauthorized) || !strings.Contains(err.Error(), "insideout login") || !strings.Contains(err.Error(), "INSIDEOUT_TOKEN") {
				t.Errorf("%s: err = %v, want ErrUnauthorized naming insideout login and INSIDEOUT_TOKEN", name, err)
			}
		}
	}
}
