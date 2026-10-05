// Package progresshub is the in-memory fan-out for roadmap progress
// updates. One Railway server instance makes process-local state the
// honest v1, the same bound as the canvas presence registry. A publish
// carries only the project id; the stream handler re-reads the tree so
// every subscriber sees the same payload as GET /roadmap/progress.
package progresshub

import "sync"

// Hub delivers a project id to every subscriber of that project whenever
// a roadmap node changes. All calls are goroutine-safe.
type Hub struct {
	mu   sync.Mutex
	subs map[string]map[chan string]struct{}
}

func New() *Hub {
	return &Hub{subs: map[string]map[chan string]struct{}{}}
}

// Subscribe delivers every publish for the project. The returned cancel
// unsubscribes and closes the channel.
func (h *Hub) Subscribe(projectID string) (<-chan string, func()) {
	ch := make(chan string, 4)
	h.mu.Lock()
	if h.subs[projectID] == nil {
		h.subs[projectID] = map[chan string]struct{}{}
	}
	h.subs[projectID][ch] = struct{}{}
	h.mu.Unlock()
	cancel := func() {
		h.mu.Lock()
		delete(h.subs[projectID], ch)
		if len(h.subs[projectID]) == 0 {
			delete(h.subs, projectID)
		}
		h.mu.Unlock()
		close(ch)
	}
	return ch, cancel
}

// Publish tells every subscriber of the project that its tree changed.
// A slow consumer misses the intermediate notice; the next one, or a
// reconnect, re-reads the current snapshot.
func (h *Hub) Publish(projectID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[projectID] {
		select {
		case ch <- projectID:
		default:
		}
	}
}
