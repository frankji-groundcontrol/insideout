package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/google/uuid"
)

// Ceilings on the passive SSE streams (presence, roadmap progress) and on the
// cursor posts that fan out to them (rv4-REL-01, SEC-08). A stream pins a
// goroutine, a connection and two registry subscriptions, about 0.4 MB
// measured (1092 open streams held 428 MB of a 512 MB container), and nothing
// bounded how many a caller could hold. A project page opens two streams, so
// the per-user ceiling is ten of a member's own tabs, and a team of a dozen
// sits well under the per-project one. The total across the server is bounded
// by users x the per-user ceiling (no global cap).
//
// The presence key carries the caller-chosen client id, but every registered
// key belongs to a live stream, so the distinct keys one user can hold are
// bounded by the per-user ceiling (each at most maxClientIDRunes long). A
// cursor post registers nothing.
const (
	maxStreamsPerUser    = 20
	maxStreamsPerProject = 100
	// maxCursorPostsPerSecond is five times what the canvas sends (its client
	// throttle is one post per 200 ms), so only a script reaches it.
	maxCursorPostsPerSecond = 25
)

// streamSlots counts the open passive streams per user and per project. The
// zero value is ready to use.
type streamSlots struct {
	mu        sync.Mutex
	byUser    map[uuid.UUID]int
	byProject map[uuid.UUID]int
}

// take reserves one stream, or reports that the user or the project is at its
// ceiling. The returned release gives the slot back.
func (c *streamSlots) take(userID, projectID uuid.UUID) (release func(), ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.byUser[userID] >= maxStreamsPerUser || c.byProject[projectID] >= maxStreamsPerProject {
		return nil, false
	}
	if c.byUser == nil {
		c.byUser, c.byProject = map[uuid.UUID]int{}, map[uuid.UUID]int{}
	}
	c.byUser[userID]++
	c.byProject[projectID]++
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		drop(c.byUser, userID)
		drop(c.byProject, projectID)
	}, true
}

func drop(counts map[uuid.UUID]int, id uuid.UUID) {
	if counts[id]--; counts[id] <= 0 {
		delete(counts, id)
	}
}

// takeStreamSlot reserves one open stream for the caller, or answers 429 and
// reports ok=false. Call it after the membership check, before any header is
// written, and defer the release.
func (s *Server) takeStreamSlot(w http.ResponseWriter, userID, projectID uuid.UUID) (release func(), ok bool) {
	release, ok = s.streams.take(userID, projectID)
	if !ok {
		httpx.WriteError(w, http.StatusTooManyRequests, "too many open streams, close a tab and retry", "APP_STREAM_LIMIT", nil)
	}
	return release, ok
}

// cursorLimiter is a per-user fixed one-second window over cursor posts. The
// zero value is ready to use.
type cursorLimiter struct {
	mu    sync.Mutex
	users map[uuid.UUID]cursorWindow
}

type cursorWindow struct {
	start time.Time
	posts int
}

// allow counts one post and reports whether it is inside the user's budget.
// ponytail: one small entry stays per user who ever posted a cursor, so the
// map is bounded by the user count; sweep stale windows if that reaches the
// tens of thousands.
func (l *cursorLimiter) allow(userID uuid.UUID, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.users == nil {
		l.users = map[uuid.UUID]cursorWindow{}
	}
	w := l.users[userID]
	if now.Sub(w.start) >= time.Second {
		w = cursorWindow{start: now}
	}
	w.posts++
	l.users[userID] = w
	return w.posts <= maxCursorPostsPerSecond
}
