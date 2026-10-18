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
