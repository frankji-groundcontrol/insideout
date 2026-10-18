package apiclient

import (
	"net/http"
	"testing"
	"time"
)

// Review-4 USER-04, found by the live proof: several processes that find the
// same access token expired all offer the same one-time refresh token. One
// wins; its rotated pair reaches the stored session a few milliseconds after
// the server answered it. A loser that looks at the store in that gap used to
// give up with "run insideout login" although the session was fine (18 of 40
// simultaneous `insideout whoami` failed against a real server, 3s access TTL).

func TestLoserOfTheRenewalRaceAdoptsTheWinnersPair(t *testing.T) {
	base, calls := sessionAPI(t, func(w http.ResponseWriter, _ string) {
		w.WriteHeader(http.StatusUnauthorized) // the winner already took the one-time token
	})
	start := time.Now()
	c := New(base)
	c.SetToken("old")
	c.SetRenewal("ref-1", Renewal{
		Reload: func() (string, string, bool) {
			if time.Since(start) < 80*time.Millisecond { // the winner's save has not landed yet
				return "old", "ref-1", true
			}
			return "new", "ref-2", true
		},
		Save: func(string, string) { t.Error("a loser adopts the stored pair; it has none of its own to save") },
	})

	if _, err := c.Whoami(); err != nil {
		t.Fatalf("Whoami: %v (the session is fine, the winner's pair was a moment away)", err)
	}
	if got, want := paths(calls()), "GET /me Bearer old | POST /auth/refresh  | GET /me Bearer new"; got != want {
		t.Errorf("calls = %s\nwant   %s", got, want)
	}
}

// A refresh token that really is dead (revoked, expired) must still end the
// session once the grace is over, not wait forever.
func TestWaitForTheWinnersPairEndsWhenNothingArrives(t *testing.T) {
	was := renewalGrace
	renewalGrace = 60 * time.Millisecond
	t.Cleanup(func() { renewalGrace = was })
	base, _ := sessionAPI(t, func(w http.ResponseWriter, _ string) { w.WriteHeader(http.StatusUnauthorized) })
	f := &file{access: "old", refresh: "ref-1"}

	begin := time.Now()
	_, err := armed(base, f).Whoami()

	if err == nil {
		t.Fatal("a dead refresh token must end the session")
	}
	if took := time.Since(begin); took < 60*time.Millisecond || took > 2*time.Second {
		t.Errorf("gave up after %v, want about the %v grace", took, renewalGrace)
	}
}
