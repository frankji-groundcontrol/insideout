package apiclient

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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
