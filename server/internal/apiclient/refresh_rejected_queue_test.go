package apiclient

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Review-4, repair round 1 (rv4-FR-F152). The MCP server arms a client for an
// in-process sign-in (SetRenewal over in-memory hooks), and tool calls can
// overlap. When the server refuses the refresh token for good, the first call to
// get the verdict disarms the client; calls that had already seen their own 401
// were queued behind it on the renewal lock and found the hooks gone.

// Four calls on one armed client are refused together and the refresh token is
// dead. Exactly one of them offers it; the others, queued behind it, end with
// the same refusal. None may call the hooks the rejection took away, and (under
// -race) none may read them while the rejection takes them away.
func TestRvApiClientCallsQueuedBehindARejectedRenewalEndWithTheRefusal(t *testing.T) {
	was := renewalGrace
	renewalGrace = 60 * time.Millisecond // the verdict comes after this; the queued calls are long in line by then
	t.Cleanup(func() { renewalGrace = was })

	const callers = 4
	var mu sync.Mutex
	arrived := 0
	allRefused := make(chan struct{}) // closes once every caller's first request has landed
	var refreshes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/refresh" {
			refreshes.Add(1)
			w.WriteHeader(http.StatusUnauthorized) // the one-time token is dead for good
			return
		}
		mu.Lock()
		if arrived++; arrived == callers {
			close(allRefused)
		}
		mu.Unlock()
		select { // so all four are refused together, whatever the scheduler does
		case <-allRefused:
		case <-time.After(5 * time.Second):
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	f := &file{access: "old", refresh: "ref-1"}
	c := armed(srv.URL, f)

	errs := make([]error, callers)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errs[i] = fmt.Errorf("panic: %v", r)
				}
			}()
			_, errs[i] = c.Whoami()
		}()
	}
	wg.Wait()

	for i, err := range errs {
		if !errors.Is(err, ErrUnauthorized) || !strings.Contains(err.Error(), "insideout login") {
			t.Errorf("call %d: err = %v, want the refusal (ErrUnauthorized telling to sign in again)", i, err)
		}
	}
	if n := refreshes.Load(); n != 1 {
		t.Errorf("the dead refresh token was offered %d times, want once", n)
	}
	if f.saved != 0 {
		t.Errorf("a rejected renewal must not write the stored session: %+v", *f)
	}
}
