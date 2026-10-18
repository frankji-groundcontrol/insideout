package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// rv4-SEC-04: a refresh token is one-use, and presenting a spent one used to
// cost only that request a 401. Whoever had rotated first (the owner, or the
// thief who copied the token) kept a live session, and the other side was
// merely refused. Reuse of a spent token now ends the whole lineage (the
// "session family": every session descended from one login), so a stolen token
// cannot outlive the owner noticing. The hazard on the other side is a client
// that legitimately presents the same token twice within moments (two CLI or
// MCP processes sharing one credentials file, two tabs, a retried request):
// that must keep refusing the loser without punishing the winner.

type refreshWorld struct {
	st *store.Store
	h  http.Handler
}

// newRefreshWorld is a real-database Server; users sign in through the real
// login route and rotate through the real refresh route.
func newRefreshWorld(t *testing.T, logs io.Writer) *refreshWorld {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	s := NewServer(st, auth.NewTokenIssuer(routeTestSecret, time.Hour),
		&config.Config{PublicURL: "http://app.test", AccessTTL: 15 * time.Minute, RefreshTTL: 720 * time.Hour},
		slog.New(slog.NewTextHandler(logs, nil)), nil, nil)
	return &refreshWorld{st: st, h: s.Handler()}
}

const refreshTestPassword = "correct horse battery staple"

func (w *refreshWorld) newUser(t *testing.T) *store.User {
	t.Helper()
	hash, err := auth.HashPassword(refreshTestPassword)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	u, err := w.st.CreateUser(context.Background(), "refresh-reuse-"+uuid.NewString()+"@test.local", hash, "refresher")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

// post sends one JSON POST and returns the status, the body, and the
// refreshToken the body carried (empty unless the call succeeded).
func (w *refreshWorld) post(path, body string) (int, string, string) {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	w.h.ServeHTTP(rec, r)
	var out struct {
		RefreshToken string `json:"refreshToken"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, rec.Body.String(), out.RefreshToken
}

// login starts a new lineage and returns its first refresh token.
func (w *refreshWorld) login(t *testing.T, u *store.User) string {
	t.Helper()
	status, body, token := w.post("/api/v1/auth/login", `{"email":"`+u.Email+`","password":"`+refreshTestPassword+`"}`)
	if status != http.StatusOK || token == "" {
		t.Fatalf("login: status %d body %s", status, body)
	}
	return token
}

func (w *refreshWorld) refresh(token string) (int, string, string) {
	return w.post("/api/v1/auth/refresh", `{"refreshToken":"`+token+`"}`)
}

// rotate refreshes and insists on success, returning the successor token.
func (w *refreshWorld) rotate(t *testing.T, token string) string {
	t.Helper()
	status, body, next := w.refresh(token)
	if status != http.StatusOK || next == "" {
		t.Fatalf("refresh: status %d body %s", status, body)
	}
	return next
}

// ageRevocations moves every revocation of the user's sessions an hour into
// the past: the rotations happened longer ago than any reuse grace.
func (w *refreshWorld) ageRevocations(t *testing.T, userID uuid.UUID) {
	t.Helper()
	if _, err := w.st.Pool.Exec(context.Background(),
		`UPDATE insideout.sessions SET revoked_at = revoked_at - interval '1 hour' WHERE user_id = $1 AND revoked_at IS NOT NULL`, userID); err != nil {
		t.Fatalf("age revocations: %v", err)
	}
}

func TestRefreshReuseAfterTheGraceEndsTheWholeFamily(t *testing.T) {
	var logs bytes.Buffer
	w := newRefreshWorld(t, &logs)
	u := w.newUser(t)
	r0 := w.login(t, u)
	r1 := w.rotate(t, r0)
	r2 := w.rotate(t, r1) // the newest token: held by the thief who rotated first, or by the owner
	w.ageRevocations(t, u.ID)

	status, replayBody, _ := w.refresh(r0)
	if status != http.StatusUnauthorized {
		t.Fatalf("replaying a long-spent token: status %d, want 401", status)
	}
	if _, unknownBody, _ := w.refresh("never-issued-" + uuid.NewString()); unknownBody != replayBody {
		t.Errorf("a replay is answered %q but an unknown token %q: the response must not tell the replayer it was noticed", replayBody, unknownBody)
	}

	if status, _, _ := w.refresh(r2); status != http.StatusUnauthorized {
		t.Fatalf("the newest token still refreshes (status %d) after a spent one was replayed: "+
			"reuse only refused the replayed request, so whoever rotated first keeps the session", status)
	}
	if !strings.Contains(logs.String(), "reuse") {
		t.Errorf("the revocation left no log line for the operator:\n%s", logs.String())
	}
}

// Within moments of a rotation the spent token is offered again by a process
// that read the credentials a moment too early. It is refused, and the pair the
// winner got stays valid; revoking here would log out every concurrent client.
func TestRefreshReplayInsideTheGraceLeavesTheWinnersPairAlive(t *testing.T) {
	w := newRefreshWorld(t, io.Discard)
	u := w.newUser(t)
	r0 := w.login(t, u)
	r1 := w.rotate(t, r0)

	if status, _, _ := w.refresh(r0); status != http.StatusUnauthorized {
		t.Fatalf("the loser of the race: status %d, want 401 (one token, one rotation)", status)
	}
	if status, body, _ := w.refresh(r1); status != http.StatusOK {
		t.Fatalf("the winner's token after the loser's replay: status %d, %s — "+
			"a replay inside the grace revoked the lineage, so two CLI/MCP processes or two tabs sharing one login would sign each other out", status, body)
	}
}

func TestConcurrentRefreshesOfOneTokenLeaveOneWinnerAlive(t *testing.T) {
	w := newRefreshWorld(t, io.Discard)
	u := w.newUser(t)
	r0 := w.login(t, u)

	const n = 16
	statuses, tokens := make([]int, n), make([]string, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			statuses[i], _, tokens[i] = w.refresh(r0)
		}()
	}
	close(start)
	wg.Wait()

	winners, winner := 0, ""
	for i, status := range statuses {
		switch status {
		case http.StatusOK:
			winners, winner = winners+1, tokens[i]
		case http.StatusUnauthorized:
		default:
			t.Fatalf("refresh %d: status %d, want 200 or 401", i, status)
		}
	}
	if winners != 1 {
		t.Fatalf("%d of %d simultaneous refreshes of one token succeeded, want exactly 1", winners, n)
	}
	if status, body, _ := w.refresh(winner); status != http.StatusOK {
		t.Fatalf("the winner's token after the race: status %d, %s — the losers' replays revoked it", status, body)
	}
}

// A user signed in on two devices has two lineages; reuse on one ends that one.
func TestReuseOnOneDeviceLeavesTheUsersOtherDevicesAlone(t *testing.T) {
	w := newRefreshWorld(t, io.Discard)
	u := w.newUser(t)
	phone, laptop := w.login(t, u), w.login(t, u)
	laptop = w.rotate(t, laptop)
	w.rotate(t, phone)
	w.ageRevocations(t, u.ID)

	if status, _, _ := w.refresh(phone); status != http.StatusUnauthorized {
		t.Fatalf("replaying the phone's spent token: status %d, want 401", status)
	}
	if status, body, _ := w.refresh(laptop); status != http.StatusOK {
		t.Fatalf("the laptop's token after reuse on the phone: status %d, %s — another device's lineage was revoked", status, body)
	}
}

// Sessions created before family_id existed carry no family. Their first
// rotation after the upgrade starts one, so a pre-upgrade token replayed later
// still ends everything that descended from it.
func TestReuseOfAPreUpgradeSessionEndsItsSuccessors(t *testing.T) {
	w := newRefreshWorld(t, io.Discard)
	u := w.newUser(t)
	legacy, legacyHash, err := auth.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	// The row shape the old code wrote: no family named.
	if _, err := w.st.Pool.Exec(context.Background(),
		`INSERT INTO insideout.sessions (user_id, token_hash, expires_at) VALUES ($1, $2, now() + interval '30 days')`,
		u.ID, legacyHash); err != nil {
		t.Fatalf("insert pre-upgrade session: %v", err)
	}
	newest := w.rotate(t, w.rotate(t, legacy))
	w.ageRevocations(t, u.ID)

	if status, _, _ := w.refresh(legacy); status != http.StatusUnauthorized {
		t.Fatalf("replaying the pre-upgrade token: status %d, want 401", status)
	}
	if status, _, _ := w.refresh(newest); status != http.StatusUnauthorized {
		t.Fatalf("a successor of the pre-upgrade session still refreshes (status %d) after its ancestor was replayed", status)
	}
}
