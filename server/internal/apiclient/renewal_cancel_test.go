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

// A renewal and a cancelled call (review-4 rv4-FR-F66): the rotated pair is
// saved even when the caller gives up mid-renewal, nothing is spent for a
// caller already gone, and WaitForRenewal waits for a renewal in flight, but
// only so long.

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
