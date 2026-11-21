package api

import (
	"net/http"
	"testing"
)

// rv5-PARA-09: a profile or workspace save that leaves avatarUrl or coverUrl
// out keeps the stored picture; null or "" clears it. Real PostgreSQL.

func TestAnAbsentAvatarOrCoverKeepsTheStoredOne(t *testing.T) {
	w := newCoverageWorld(t)
	const pic = "https://img.example/a.png"

	me := func(body map[string]any) any {
		t.Helper()
		status, resp := w.send(t, w.member, "PATCH", "/api/v1/me", body)
		expect(t, "PATCH /me", status, resp, http.StatusOK)
		return decodeAs[map[string]any](t, resp)["avatarUrl"]
	}
	ws := func(body map[string]any) any {
		t.Helper()
		status, resp := w.send(t, w.owner, "PATCH", w.ws(), body)
		expect(t, "PATCH workspace", status, resp, http.StatusOK)
		return decodeAs[map[string]any](t, resp)["coverUrl"]
	}

	for _, clear := range []any{nil, ""} {
		if got := me(map[string]any{"username": "member", "avatarUrl": pic}); got != pic {
			t.Fatalf("avatar set: %v", got)
		}
		if got := me(map[string]any{"username": "member", "bio": "kept"}); got != pic {
			t.Errorf("a save without avatarUrl left %v, want the stored avatar", got)
		}
		if got := me(map[string]any{"username": "member", "avatarUrl": clear}); got != nil {
			t.Errorf("avatarUrl %#v left %v, want it cleared", clear, got)
		}

		if got := ws(map[string]any{"title": "Route tests", "coverUrl": pic}); got != pic {
			t.Fatalf("cover set: %v", got)
		}
		if got := ws(map[string]any{"title": "Route tests", "description": "kept"}); got != pic {
			t.Errorf("a save without coverUrl left %v, want the stored cover", got)
		}
		if got := ws(map[string]any{"title": "Route tests", "coverUrl": clear}); got != nil {
			t.Errorf("coverUrl %#v left %v, want it cleared", clear, got)
		}
	}
}
