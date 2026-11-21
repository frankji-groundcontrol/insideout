package api

import (
	"net/http"
	"testing"
)

// rv5 wave 2 addendum: a project view says whether the caller may change the
// plan itself (a root node, a rebuild), so the web offers root-level "New
// item" only where it would succeed. Real PostgreSQL.
func TestAProjectViewSaysWhetherTheCallerWritesThePlan(t *testing.T) {
	w := newRouteWorld(t)
	for _, c := range []struct {
		who  routeUser
		name string
		want bool
	}{{w.owner, "owner", true}, {w.member, "member", false}} {
		one := decodeAs[map[string]any](t, w.must(t, c.who, "GET", w.proj(), "", http.StatusOK))
		if got, ok := one["canWritePlan"].(bool); !ok || got != c.want {
			t.Errorf("%s GET project: canWritePlan = %v, want %v", c.name, one["canWritePlan"], c.want)
		}
		list := decodeAs[[]map[string]any](t, w.must(t, c.who, "GET", w.ws()+"/projects", "", http.StatusOK))
		if len(list) != 1 {
			t.Fatalf("%s project list: %d projects, want 1", c.name, len(list))
		}
		if got, ok := list[0]["canWritePlan"].(bool); !ok || got != c.want {
			t.Errorf("%s project list: canWritePlan = %v, want %v", c.name, list[0]["canWritePlan"], c.want)
		}
	}
	created := decodeAs[map[string]any](t, w.must(t, w.member, "POST", w.ws()+"/projects", `{"title":"Mine"}`, http.StatusCreated))
	if created["canWritePlan"] != true {
		t.Errorf("a member's own new project: canWritePlan = %v, want true", created["canWritePlan"])
	}
	edited := decodeAs[map[string]any](t, w.must(t, w.owner, "PATCH", w.proj(), `{"title":"Renamed","status":"active"}`, http.StatusOK))
	if edited["canWritePlan"] != true {
		t.Errorf("owner's PATCH response: canWritePlan = %v, want true", edited["canWritePlan"])
	}
}
