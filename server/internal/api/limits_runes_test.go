package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/google/uuid"
)

// COMP-01: a field limit counts characters, not bytes. 67 Chinese characters
// are 201 bytes, and a byte check refused them with "max 200 characters". Every
// capped field below must take limit/3+1 characters (more bytes than the limit,
// fewer characters) and exactly limit characters of three-byte text, and must
// refuse limit+1 with its own limit message.

// cjk is n copies of one three-byte character.
func cjk(n int) string { return strings.Repeat("字", n) }

func jsonBody(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// send makes one JSON request as the given caller.
func (w *routeWorld) send(t *testing.T, as routeUser, method, path string, body map[string]any) (int, string) {
	t.Helper()
	return w.call(as, method, path, jsonBody(t, body))
}

// idea inserts a fresh idea authored by the owner and returns its id.
func (w *routeWorld) idea(t *testing.T) string {
	t.Helper()
	idea, err := w.st.CreateIdea(context.Background(), w.owner.ID, w.project.WorkspaceID, "seed idea", "seed")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	return idea.ID.String()
}

// prd converts a fresh idea and returns the PRD id.
func (w *routeWorld) prd(t *testing.T) string {
	t.Helper()
	prd, _, err := w.st.ConvertIdea(context.Background(), w.owner.ID, uuid.MustParse(w.idea(t)))
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}
	return prd.ID.String()
}

type runeLimit struct {
	name  string
	limit int
	// msg is what the handler answers when the value is too long.
	msg string
	// ok: an accepted value must come back 2xx. False only for register,
	// whose signup policy (invitation-only by default) answers 403 after the
	// length check has passed.
	ok   bool
	send func(t *testing.T, w *routeWorld, v string) (int, string)
}

func runeLimits() []runeLimit {
	const title = "max 200 characters"
	pid := func(w *routeWorld) string { return w.project.ID.String() }
	ws := func(w *routeWorld) string { return w.project.WorkspaceID.String() }
	node := func(t *testing.T, w *routeWorld) string { return w.roadmapNode(t, "seed node") }
	doneNode := func(t *testing.T, w *routeWorld) string {
		return w.roadmapNode(t, "seed done node", `{"status":"in_progress"}`, `{"status":"done"}`)
	}
	owner := func(t *testing.T, w *routeWorld, method, path string, body map[string]any) (int, string) {
		return w.send(t, w.owner, method, path, body)
	}
	return []runeLimit{
		{"idea create title", 200, title, true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "POST", "/api/v1/workspaces/"+ws(w)+"/ideas", map[string]any{"title": v, "content": "c"})
		}},
		{"idea update title", 200, title, true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "PATCH", "/api/v1/ideas/"+w.idea(t), map[string]any{"title": v, "content": "c"})
		}},
		{"project create title", 200, title, true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "POST", "/api/v1/workspaces/"+ws(w)+"/projects", map[string]any{"title": v})
		}},
		{"project update title", 200, title, true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "PATCH", "/api/v1/projects/"+pid(w), map[string]any{"title": v, "status": "active"})
		}},
		{"workspace create title", 200, title, true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "POST", "/api/v1/workspaces", map[string]any{"title": v})
		}},
		{"workspace update title", 200, title, true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "PATCH", "/api/v1/workspaces/"+ws(w), map[string]any{"title": v})
		}},
		{"prd update title", 200, "1-200 characters", true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "PATCH", "/api/v1/prds/"+w.prd(t), map[string]any{"title": v})
		}},
		{"prd commit name", 200, title, true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "POST", "/api/v1/prds/"+w.prd(t)+"/commit", map[string]any{"name": v, "primaryAudience": "decision"})
		}},
		{"roadmap node create title", 200, title, true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "POST", "/api/v1/projects/"+pid(w)+"/roadmap", map[string]any{"title": v})
		}},
		{"roadmap node update title", 200, title, true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "PATCH", "/api/v1/roadmap/"+node(t, w), map[string]any{"title": v})
		}},
		{"validation observation detail", 2000, "detail is required (max 2000 characters)", true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "POST", "/api/v1/roadmap/"+node(t, w)+"/evidence", map[string]any{"detail": v, "observedAt": "2026-10-02"})
		}},
		{"validation observation source url", 2000, "sourceUrl is too long (max 2000 characters)", true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "POST", "/api/v1/roadmap/"+node(t, w)+"/evidence", map[string]any{"detail": "seen", "sourceUrl": v, "observedAt": "2026-10-02"})
		}},
		{"validation confirm reason", 2000, "reason is required (max 2000 characters)", true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "POST", "/api/v1/roadmap/"+doneNode(t, w)+"/validate", map[string]any{"reason": v})
		}},
		{"agent checkpoint summary", 2000, "summary is required (max 2000 characters)", true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "POST", "/api/v1/agent/checkpoint", map[string]any{"projectId": pid(w), "summary": v})
		}},
		{"agent proposal summary", 2000, "summary is required (max 2000 characters)", true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "POST", "/api/v1/agent/propose", map[string]any{"projectId": pid(w), "kind": "structure", "summary": v})
		}},
		{"agent proposal item title", 200, "item title is required (max 200 characters)", true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "POST", "/api/v1/agent/propose", map[string]any{
				"projectId": pid(w), "kind": "structure", "summary": "s", "items": []map[string]any{{"title": v}},
			})
		}},
		{"register username", 60, "username is required (max 60 characters)", false, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return w.send(t, routeUser{}, "POST", "/api/v1/auth/register", map[string]any{
				"email": "rune-limit-" + uuid.NewString() + "@example.com", "password": "correct-horse-battery", "username": v,
			})
		}},
		{"profile username", 60, "username is required (max 60 characters)", true, func(t *testing.T, w *routeWorld, v string) (int, string) {
			return owner(t, w, "PATCH", "/api/v1/me", map[string]any{"username": v})
		}},
	}
}

func TestFieldLimitsCountCharactersNotBytes(t *testing.T) {
	w := newRouteWorld(t)
	for _, tc := range runeLimits() {
		tooLongAnswer := func(code int, body string) bool {
			return code == http.StatusBadRequest && strings.Contains(body, tc.msg)
		}
		accept := func(t *testing.T, n int) {
			t.Helper()
			v := cjk(n)
			code, body := tc.send(t, w, v)
			if tooLongAnswer(code, body) {
				t.Fatalf("%d characters (%d bytes) refused as too long (limit %d): %s", utf8.RuneCountInString(v), len(v), tc.limit, body)
			}
			if tc.ok && code/100 != 2 {
				t.Fatalf("%d characters: status %d, want 2xx: %s", n, code, body)
			}
		}
		t.Run(tc.name, func(t *testing.T) {
			t.Run("more bytes than the limit, fewer characters", func(t *testing.T) { accept(t, tc.limit/3+1) })
			t.Run("exactly the limit in characters", func(t *testing.T) { accept(t, tc.limit) })
			t.Run("one character over the limit", func(t *testing.T) {
				code, body := tc.send(t, w, cjk(tc.limit+1))
				if !tooLongAnswer(code, body) {
					t.Fatalf("%d characters: status %d body %s, want 400 with %q", tc.limit+1, code, body, tc.msg)
				}
			})
		})
	}
}

// tooLong is the one place the character count lives.
func TestTooLongCountsRunes(t *testing.T) {
	for _, tc := range []struct {
		s    string
		n    int
		want bool
	}{
		{"", 0, false},
		{"abc", 3, false},
		{"abcd", 3, true},
		{cjk(67), 200, false}, // 201 bytes
		{cjk(200), 200, false},
		{cjk(201), 200, true},
		{"héllo", 5, false}, // 6 bytes
	} {
		if got := tooLong(tc.s, tc.n); got != tc.want {
			t.Errorf("tooLong(%d runes, %d) = %v, want %v", utf8.RuneCountInString(tc.s), tc.n, got, tc.want)
		}
	}
}
