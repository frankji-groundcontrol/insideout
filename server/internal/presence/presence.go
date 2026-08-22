// Package presence is the in-memory who-is-here registry behind the
// collaborative canvas. One Railway server instance makes process-local
// state the honest v1; the TTL prune drops stale tabs without goodbye.
package presence

import (
	"sync"
	"time"
)

// Entry is one live viewer session (one browser tab).
type Entry struct {
	SessionID string    `json:"sessionId"`
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	LastSeen  time.Time `json:"-"`
}

// Registry tracks per-project sessions and pushes change snapshots to
// subscribers. All calls are goroutine-safe.
type Registry struct {
	mu       sync.Mutex
	projects map[string]map[string]Entry
	subs     map[string]map[chan []Entry]struct{}
	ttl      time.Duration
	now      func() time.Time
}

func New(ttl time.Duration, now func() time.Time) *Registry {
	if now == nil {
		now = time.Now
	}
	return &Registry{
		projects: map[string]map[string]Entry{},
		subs:     map[string]map[chan []Entry]struct{}{},
		ttl:      ttl, now: now,
	}
}

// Touch upserts a session heartbeat and returns the pruned snapshot.
func (r *Registry) Touch(projectID, sessionID, userID, name string) []Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.projects[projectID]
	if !ok {
		m = map[string]Entry{}
		r.projects[projectID] = m
	}
	m[sessionID] = Entry{SessionID: sessionID, UserID: userID, Name: name, LastSeen: r.now()}
	snap := r.pruneLocked(projectID)
	r.broadcastLocked(projectID, snap)
	return snap
}

// Leave removes a session (SSE disconnect) and broadcasts.
func (r *Registry) Leave(projectID, sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.projects[projectID]; ok {
		delete(m, sessionID)
		if len(m) == 0 {
			delete(r.projects, projectID)
		}
	}
	snap := r.pruneLocked(projectID)
	r.broadcastLocked(projectID, snap)
}

// List returns the pruned snapshot without side effects.
func (r *Registry) List(projectID string) []Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.pruneLocked(projectID)
}

// Subscribe delivers every changed snapshot for the project; the
// returned cancel func unsubscribes.
func (r *Registry) Subscribe(projectID string) (<-chan []Entry, func()) {
	ch := make(chan []Entry, 4)
	r.mu.Lock()
	if r.subs[projectID] == nil {
		r.subs[projectID] = map[chan []Entry]struct{}{}
	}
	r.subs[projectID][ch] = struct{}{}
	r.mu.Unlock()
	cancel := func() {
		r.mu.Lock()
		delete(r.subs[projectID], ch)
		if len(r.subs[projectID]) == 0 {
			delete(r.subs, projectID)
		}
		r.mu.Unlock()
		close(ch)
	}
	return ch, cancel
}

func (r *Registry) pruneLocked(projectID string) []Entry {
	m, ok := r.projects[projectID]
	if !ok {
		return []Entry{}
	}
	cutoff := r.now().Add(-r.ttl)
	for id, e := range m {
		if e.LastSeen.Before(cutoff) {
			delete(m, id)
		}
	}
	out := make([]Entry, 0, len(m))
	for _, e := range m {
		out = append(out, e)
	}
	// Deterministic order for stable diffs.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].SessionID < out[j-1].SessionID; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func (r *Registry) broadcastLocked(projectID string, snap []Entry) {
	for ch := range r.subs[projectID] {
		select {
		case ch <- snap:
		default: // slow consumer: drop; the next change resyncs
		}
	}
}
