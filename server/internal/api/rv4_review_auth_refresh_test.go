package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// rv4-FR-F100, F108, F159, F24: the 10 s refresh-reuse grace keeps the
// winner's session family alive on the server, but the request that lost the
// race was still answered with Set-Cookie deletions of both auth cookies. A
// browser applies each response's headers in the order they arrive, so a loser
// that lands after the winner deletes the pair the winner just set: two tabs on
// one cookie jar signed each other out, which is what the grace exists to
// prevent. The grace tests in auth_refresh_reuse_test.go carry the token in the
// JSON body and assert only server state, so they cannot see the cookies. The
// tests below play the browser: a jar that applies Set-Cookie in arrival
// order. The status stays 401 (the CLI's renewal wait and the Flutter client's
// refreshEndsSession are written against it); what changes is that a loser
// inside the grace touches nothing.

// rvAuthRefreshJar is the part of a browser that matters here: the cookies it
// holds, and what a response's Set-Cookie headers do to them (a Max-Age of
// zero deletes).
type rvAuthRefreshJar map[string]string

func (j rvAuthRefreshJar) clone() rvAuthRefreshJar {
	c := make(rvAuthRefreshJar, len(j))
	for name, value := range j {
		c[name] = value
	}
	return c
}

func (j rvAuthRefreshJar) apply(resp *http.Response) {
	for _, c := range resp.Cookies() {
		if c.MaxAge < 0 {
			delete(j, c.Name)
			continue
		}
		j[c.Name] = c.Value
	}
}

// holds says which auth cookies the jar has, for failure messages (never the values).
func (j rvAuthRefreshJar) holds() string {
	return fmt.Sprintf("access cookie %t, refresh cookie %t", j[accessCookieName] != "", j[refreshCookieName] != "")
}

// rvAuthRefreshLogin signs in through the real route and returns the jar the
// browser holds afterwards.
func rvAuthRefreshLogin(t *testing.T, w *refreshWorld, u *store.User) rvAuthRefreshJar {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"email":"`+u.Email+`","password":"`+refreshTestPassword+`"}`))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	w.h.ServeHTTP(rec, r)
	resp := rec.Result()
	jar := rvAuthRefreshJar{}
	jar.apply(resp)
	if resp.StatusCode != http.StatusOK || jar[accessCookieName] == "" || jar[refreshCookieName] == "" {
		t.Fatalf("login: status %d, browser holds %s", resp.StatusCode, jar.holds())
	}
	return jar
}

// rvAuthRefreshPost is POST /auth/refresh as a browser sends it: the jar's
// cookies and no body.
func rvAuthRefreshPost(w *refreshWorld, jar rvAuthRefreshJar) *http.Response {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	for name, value := range jar {
		r.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	rec := httptest.NewRecorder()
	w.h.ServeHTTP(rec, r)
	return rec.Result()
}

// rvAuthRefreshRequireBareRefusal fails unless resp is what a loser inside the
// grace gets: 401, and no auth cookie set or deleted.
func rvAuthRefreshRequireBareRefusal(t *testing.T, what string, resp *http.Response) {
	t.Helper()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("%s: status %d, want 401 (one token, one rotation; the clients are written against the status)", what, resp.StatusCode)
	}
	for _, c := range resp.Cookies() {
		if c.Name == accessCookieName || c.Name == refreshCookieName {
			t.Errorf("%s: the 401 sets or deletes %s (Max-Age %d): a browser applies it after the winner's response and takes the winner's fresh pair down with it", what, c.Name, c.MaxAge)
		}
	}
}

// Sequential form of the race: tab 2 sends the token tab 1 already spent,
// moments after the rotation (a tab that read its cookie a beat too early, or
// a retry whose first answer was lost).
func TestRvAuthRefreshReplayInsideTheGraceLeavesTheCookiesAlone(t *testing.T) {
	w := newRefreshWorld(t, io.Discard)
	jar := rvAuthRefreshLogin(t, w, w.newUser(t))
	r0 := jar[refreshCookieName]
	tab1, tab2 := jar.clone(), jar.clone() // two tabs, one jar: both send r0

	won := rvAuthRefreshPost(w, tab1)
	if won.StatusCode != http.StatusOK {
		t.Fatalf("the winner: status %d, want 200", won.StatusCode)
	}
	browser := jar.clone()
	browser.apply(won)
	r1 := browser[refreshCookieName]
	if r1 == "" || r1 == r0 {
		t.Fatalf("the winner's response did not set a fresh refresh cookie")
	}

	lost := rvAuthRefreshPost(w, tab2)
	rvAuthRefreshRequireBareRefusal(t, "the replay inside the grace", lost)
	browser.apply(lost) // arrives after the winner's
	if browser[refreshCookieName] != r1 || browser[accessCookieName] == "" {
		t.Fatalf("after the loser's response the browser holds %s, want the winner's pair: "+
			"the grace kept the family alive on the server, but the loser's 401 wiped the cookies", browser.holds())
	}
	if next := rvAuthRefreshPost(w, browser); next.StatusCode != http.StatusOK {
		t.Fatalf("the winner's pair after the loser's replay: status %d, want 200", next.StatusCode)
	}
}

// rvAuthRefreshWaitForRowLockWaiter blocks until RotateSession's revoking
// UPDATE sits waiting on a row lock another transaction holds.
func rvAuthRefreshWaitForRowLockWaiter(t *testing.T, w *refreshWorld) {
	t.Helper()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		var waiting int
		if err := w.st.Pool.QueryRow(context.Background(), `
			SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database() AND wait_event_type = 'Lock'
			  AND query LIKE 'UPDATE insideout.sessions SET revoked_at = now() WHERE id%'`).Scan(&waiting); err != nil {
			t.Fatalf("poll for the parked refresh: %v", err)
		}
		if waiting > 0 {
			return
		}
	}
	t.Fatal("the refresh never reached RotateSession's row lock")
}

// The ErrConflict form, forced rather than hoped for: the refresh finds its
// session live, then waits in RotateSession on a row lock while another
// transaction (the winner) revokes that session and stores its successor. The
// refresh wakes to zero rows and loses. Real database, no hooks in the handler.
func TestRvAuthRefreshLosingTheRotationRaceLeavesTheCookiesAlone(t *testing.T) {
	w := newRefreshWorld(t, io.Discard)
	u := w.newUser(t)
	jar := rvAuthRefreshLogin(t, w, u)
	ctx := context.Background()

	tx, err := w.st.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)
	var sessID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM insideout.sessions WHERE token_hash = $1 FOR UPDATE`,
		auth.HashRefreshToken(jar[refreshCookieName])).Scan(&sessID); err != nil {
		t.Fatalf("lock the session row: %v", err)
	}

	loser := make(chan *http.Response, 1)
	go func() { loser <- rvAuthRefreshPost(w, jar.clone()) }()
	rvAuthRefreshWaitForRowLockWaiter(t, w)

	// The winner commits: the old session revoked, its successor stored.
	r1, r1Hash, err := auth.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE insideout.sessions SET revoked_at = now() WHERE id = $1`, sessID); err != nil {
		t.Fatalf("revoke the old session: %v", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO insideout.sessions (user_id, token_hash, expires_at, family_id) VALUES ($1, $2, now() + interval '1 hour', $3)`,
		u.ID, r1Hash, sessID); err != nil {
		t.Fatalf("store the successor: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var lost *http.Response
	select {
	case lost = <-loser:
	case <-time.After(10 * time.Second):
		t.Fatal("the refresh never returned after the winner committed")
	}
	rvAuthRefreshRequireBareRefusal(t, "the loser of the rotation race", lost)

	browser := jar.clone() // the winner's response landed first
	browser[refreshCookieName] = r1
	browser.apply(lost)
	if browser[refreshCookieName] != r1 {
		t.Fatalf("after the loser's response the browser holds %s, want the winner's refresh cookie", browser.holds())
	}
	if next := rvAuthRefreshPost(w, browser); next.StatusCode != http.StatusOK {
		t.Fatalf("the winner's token after the loser's refusal: status %d, want 200", next.StatusCode)
	}
}

// The concurrent form: several tabs fire at once with one jar. Exactly one
// wins; whichever branch each loser took (not found inside the grace, or lost
// at RotateSession), its answer is a bare 401. The winner's response is applied
// first and every loser's after it, the worst order the network can deliver.
func TestRvAuthRefreshConcurrentTabsOnOneJarKeepTheWinnersPair(t *testing.T) {
	w := newRefreshWorld(t, io.Discard)
	jar := rvAuthRefreshLogin(t, w, w.newUser(t))

	const tabs = 8
	resps := make([]*http.Response, tabs)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range resps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			resps[i] = rvAuthRefreshPost(w, jar.clone())
		}()
	}
	close(start)
	wg.Wait()

	winner := -1
	for i, resp := range resps {
		switch resp.StatusCode {
		case http.StatusOK:
			if winner >= 0 {
				t.Fatalf("tabs %d and %d both rotated the one token", winner, i)
			}
			winner = i
		case http.StatusUnauthorized:
		default:
			t.Fatalf("tab %d: status %d, want 200 or 401", i, resp.StatusCode)
		}
	}
	if winner < 0 {
		t.Fatal("no tab won the rotation")
	}

	browser := jar.clone()
	browser.apply(resps[winner])
	fresh := browser[refreshCookieName]
	for i, resp := range resps {
		if i != winner {
			rvAuthRefreshRequireBareRefusal(t, fmt.Sprintf("loser tab %d", i), resp)
			browser.apply(resp)
		}
	}
	if browser[refreshCookieName] != fresh || browser[accessCookieName] == "" {
		t.Fatalf("after every loser's response the browser holds %s, want the winner's pair", browser.holds())
	}
	if next := rvAuthRefreshPost(w, browser); next.StatusCode != http.StatusOK {
		t.Fatalf("the winner's pair after the race: status %d, want 200", next.StatusCode)
	}
}

// Outside the grace nothing changes: a token spent long ago and offered again
// is a copy in someone else's hands. The replay is answered like an unknown
// token (401, both cookies cleared) and the family ends.
func TestRvAuthRefreshReplayAfterTheGraceStillEndsTheFamilyAndClearsTheCookies(t *testing.T) {
	w := newRefreshWorld(t, io.Discard)
	u := w.newUser(t)
	jar := rvAuthRefreshLogin(t, w, u)
	replayer := jar.clone() // r0, about to be spent

	won := rvAuthRefreshPost(w, jar)
	if won.StatusCode != http.StatusOK {
		t.Fatalf("the rotation: status %d, want 200", won.StatusCode)
	}
	jar.apply(won) // r1 is the live token
	w.ageRevocations(t, u.ID)

	lost := rvAuthRefreshPost(w, replayer)
	if lost.StatusCode != http.StatusUnauthorized {
		t.Fatalf("a replay after the grace: status %d, want 401", lost.StatusCode)
	}
	replayer.apply(lost)
	if len(replayer) != 0 {
		t.Errorf("a replay after the grace must clear the cookies it was sent; the browser still holds %s", replayer.holds())
	}
	if next := rvAuthRefreshPost(w, jar); next.StatusCode != http.StatusUnauthorized {
		t.Errorf("the newest token still refreshes (status %d) after a long-spent one was replayed: the family must end", next.StatusCode)
	}
}

// A cookie that names no spent token is junk and is cleared, as before.
func TestRvAuthRefreshTokenThatIsNotASpentOneIsStillCleared(t *testing.T) {
	w := newRefreshWorld(t, io.Discard)
	u := w.newUser(t)
	expired, expiredHash, err := auth.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if _, err := w.st.Pool.Exec(context.Background(),
		`INSERT INTO insideout.sessions (user_id, token_hash, expires_at) VALUES ($1, $2, now() - interval '1 hour')`,
		u.ID, expiredHash); err != nil {
		t.Fatalf("insert expired session: %v", err)
	}
	for name, token := range map[string]string{
		"an unknown token":                        "never-issued-" + uuid.NewString(),
		"an expired token that was never revoked": expired,
	} {
		browser := rvAuthRefreshJar{accessCookieName: "stale-access", refreshCookieName: token}
		resp := rvAuthRefreshPost(w, browser)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s: status %d, want 401", name, resp.StatusCode)
		}
		browser.apply(resp)
		if len(browser) != 0 {
			t.Errorf("%s: the response must clear both cookies; the browser still holds %s", name, browser.holds())
		}
	}
}
