package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// REL-01 / SEC-08: nothing bounded how many passive SSE streams (presence,
// roadmap progress) one caller, or one project, could hold open. Each pins a
// goroutine, a connection and two registry subscriptions; the review measured
// 1092 open streams at 428 MB RSS against a 512 MB container. The server must
// turn the excess away with a 429, and rate-limit the cursor posts, which fan
// out to every viewer of the canvas. DB-gated: both stream handlers check
// project membership against real RLS.

// The ceilings are pinned to the numbers the w3-stream-bounds changelog records
// (20 streams per member, 100 per project, 25 cursor posts a second), never to
// the constants under test and never to a range: either follows stream_caps.go
// to whatever it is edited to, and a range lets a ceiling of 49 or 130 through
// (rv4-FR-F137). Changing a ceiling means changing the docs and these pins.
const (
	rvGuardsStreamsPerUser    = 20
	rvGuardsStreamsPerProject = 100
	rvGuardsCursorPostsPerSec = 25
)

// tryOpenSSE opens a stream and returns its status, so a refusal is data and
// not a fatal error. A 200 stream stays open until the test ends.
func tryOpenSSE(t *testing.T, ts *httptest.Server, token, path string) int {
	t.Helper()
	code, _ := openSSEWithCancel(t, ts, token, path)
	return code
}

// openSSEWithCancel is tryOpenSSE that also returns the func that closes the
// stream, the way a tab does.
func openSSEWithCancel(t *testing.T, ts *httptest.Server, token, path string) (int, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+path, nil)
	if err != nil {
		t.Fatalf("build stream request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp.StatusCode, cancel
}

func (f *streamFixture) progressPath() string {
	return "/api/v1/projects/" + f.proj.ID.String() + "/roadmap/progress/stream"
}

// tally opens one stream and counts it as accepted or refused; any other
// status is a bug in the test or the server.
func tally(t *testing.T, code int, accepted, refused *int) {
	t.Helper()
	switch code {
	case http.StatusOK:
		*accepted++
	case http.StatusTooManyRequests:
		*refused++
	default:
		t.Fatalf("stream status %d, want 200 or 429", code)
	}
}

func TestStreamCaps_OneUserIsRefusedPastACeiling(t *testing.T) {
	fx := newStreamFixture(t)
	const attempts = 150 // past any per-project ceiling, so only a per-user cap can stop this
	accepted, refused, presenceOpen := 0, 0, 0
	for i := 0; i < attempts; i++ {
		path := fx.presencePath(fmt.Sprintf("?client=tab-%d", i)) // a fresh client id every time
		if i%2 == 1 {
			path = fx.progressPath()
		}
		code := tryOpenSSE(t, fx.ts, fx.tokenA, path)
		tally(t, code, &accepted, &refused)
		if i%2 == 0 && code == http.StatusOK {
			presenceOpen++
		}
	}

	if refused == 0 {
		t.Errorf("one member opened %d streams and none was refused: nothing bounds a caller's open streams", attempts)
	}
	if accepted != rvGuardsStreamsPerUser {
		t.Errorf("one member holds %d of %d streams, want exactly the per-user ceiling of %d", accepted, attempts, rvGuardsStreamsPerUser)
	}
	// The presence key carries the caller-chosen client id, and every
	// registered key belongs to a live stream: one session per open presence
	// stream, none left behind by a refused one.
	own := 0
	for _, e := range fx.listPresence(t, fx.tokenA) {
		if e.UserID == fx.a.ID.String() {
			own++
		}
	}
	if own != presenceOpen {
		t.Errorf("one member has %d presence sessions for %d open presence streams, want one each", own, presenceOpen)
	}
}

func TestStreamCaps_AProjectIsRefusedPastACeilingAcrossUsers(t *testing.T) {
	fx := newStreamFixture(t)
	ctx := context.Background()
	st := fx.srv.store
	ws, err := st.GetWorkspaceForMember(ctx, fx.proj.WorkspaceID, fx.a.ID)
	if err != nil {
		t.Fatalf("read the workspace invite code: %v", err)
	}
	tokens := []string{fx.tokenA, fx.tokenB}
	for i := 0; i < 7; i++ {
		u, err := st.CreateUser(ctx, fmt.Sprintf("crowd-%d-%s@test.local", i, uuid.NewString()), "x", "crowd")
		if err != nil {
			t.Fatalf("create user: %v", err)
		}
		if _, err := st.JoinWorkspace(ctx, u.ID, ws.Code); err != nil {
			t.Fatalf("join workspace: %v", err)
		}
		tok, err := fx.srv.tokens.MintAccessToken(u.ID)
		if err != nil {
			t.Fatalf("mint token: %v", err)
		}
		tokens = append(tokens, tok)
	}

	// Nine members at 15 streams each stay under a per-user ceiling of 20, so
	// only a per-project ceiling can stop the crowd.
	const perUser = 15
	accepted, refused := 0, 0
	for round := 0; round < perUser; round++ {
		for _, tok := range tokens {
			tally(t, tryOpenSSE(t, fx.ts, tok, fx.progressPath()), &accepted, &refused)
		}
	}

	if refused == 0 {
		t.Errorf("%d members opened %d streams on one project and none was refused: nothing bounds a project's open streams", len(tokens), perUser*len(tokens))
	}
	if accepted != rvGuardsStreamsPerProject {
		t.Errorf("the project holds %d streams, want exactly the per-project ceiling of %d", accepted, rvGuardsStreamsPerProject)
	}
}

func TestStreamCaps_CursorPostsAreRateLimitedPerUser(t *testing.T) {
	fx := newStreamFixture(t)
	var mu sync.Mutex
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	fx.srv.clock = func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	post := func(token string, i int) int {
		resp := doJSON(t, fx.ts, token, http.MethodPost, "/api/v1/projects/"+fx.proj.ID.String()+"/cursor?client=c"+strconv.Itoa(i), `{"x":1,"y":2}`)
		resp.Body.Close()
		return resp.StatusCode
	}

	first := 0 // the first refused post, with the clock held inside a single instant
	for i := 1; i <= 300 && first == 0; i++ {
		switch code := post(fx.tokenA, i); code {
		case http.StatusNoContent:
		case http.StatusTooManyRequests:
			first = i
		default:
			t.Fatalf("cursor post %d: status %d, want 204 or 429", i, code)
		}
	}
	if first == 0 {
		t.Fatal("300 cursor posts in one instant were all accepted: nothing limits how fast one member fans out to every viewer of the canvas")
	}
	if first <= 5 {
		t.Fatalf("post %d was refused: the canvas sends about 5 a second and must never be limited at that rate", first)
	}
	if first != rvGuardsCursorPostsPerSec+1 {
		t.Fatalf("post %d was the first refused, want post %d: the budget is %d posts a second", first, rvGuardsCursorPostsPerSec+1, rvGuardsCursorPostsPerSec)
	}
	if code := post(fx.tokenB, 1); code != http.StatusNoContent {
		t.Fatalf("another member's cursor post = %d, want 204: the limit is per user", code)
	}
	mu.Lock()
	now = now.Add(2 * time.Second)
	mu.Unlock()
	if code := post(fx.tokenA, first+1); code != http.StatusNoContent {
		t.Fatalf("cursor post after the window = %d, want 204: the limit must lift", code)
	}
}

// A slot is only worth a ceiling if it comes back: a closed tab must make room
// for the next one, or every reconnect would count against the user forever.
func TestStreamCaps_ClosingAStreamFreesItsSlot(t *testing.T) {
	fx := newStreamFixture(t)
	var open []context.CancelFunc
	for i := 0; i < 100; i++ { // far past the per-user ceiling
		code, cancel := openSSEWithCancel(t, fx.ts, fx.tokenA, fx.progressPath())
		if code == http.StatusTooManyRequests {
			break
		}
		open = append(open, cancel)
	}
	if len(open) != rvGuardsStreamsPerUser {
		t.Fatalf("opened %d streams before the first refusal, want the per-user ceiling of %d", len(open), rvGuardsStreamsPerUser)
	}

	open[0]() // one tab closes
	deadline := time.Now().Add(3 * time.Second)
	for {
		if code, _ := openSSEWithCancel(t, fx.ts, fx.tokenA, fx.progressPath()); code == http.StatusOK {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("a stream closed 3s ago but the ceiling still counts it: the slot was never released")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
