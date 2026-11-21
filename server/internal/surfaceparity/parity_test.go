package surfaceparity

import (
	"slices"
	"sort"
	"strings"
	"testing"
)

// TestEveryRequestFieldIsReachable walks every JSON field an API handler
// decodes and fails unless the Go client, the CLI and the MCP tools each reach
// it, or omissions (omissions_test.go) names it for that surface with a reason.
// A stale omission (the field is reachable after all) fails too, and so does
// an MCP omission that disagrees with agentRoutePolicy in agent_scope.go.
func TestEveryRequestFieldIsReachable(t *testing.T) {
	api := loadAPI(t)
	routes := api.routes(t)
	fields := api.requestFields(t)
	policy := api.agentPolicy(t)
	sends := clientSends(t, routes)
	cli := loadSurface(t, "cli", "cmd/insideout")
	mcp := loadSurface(t, "mcp", "internal/mcpserver")

	reached := map[string]func(deps []dep) bool{
		"client": func(deps []dep) bool { return len(deps) > 0 },
		"cli":    func(deps []dep) bool { return slices.ContainsFunc(deps, cli.reaches) },
		"mcp":    func(deps []dep) bool { return slices.ContainsFunc(deps, mcp.reaches) },
	}
	used := map[int]bool{}
	var gaps []string
	for _, route := range sortedKeys(fields) {
		for _, key := range fields[route] {
			for _, surf := range []string{"client", "cli", "mcp"} {
				i := findOmission(route, key, surf)
				ok := reached[surf](depsOf(sends[route], key))
				switch {
				case !ok && i < 0:
					gaps = append(gaps, route+" "+key+": not reachable from "+surf)
				case ok && i >= 0:
					t.Errorf("%s %s: reachable from %s, but omissions[%d] still lists it; drop the entry", route, key, surf, i)
				case i >= 0:
					used[i] = true
				}
			}
		}
	}
	for _, g := range gaps {
		t.Errorf("%s: wire it, or list it in omissions (omissions_test.go) with a reason", g)
	}
	for i, o := range omissions {
		if !used[i] {
			t.Errorf("omissions[%d] (%s %s on %s) names no unreachable request field", i, o.route, o.key, o.surfaces)
		}
		if o.humanOnly != (policy[o.route] == "humanOnly") {
			t.Errorf("omissions[%d] (%s): humanOnly=%v, but agent_scope.go has it as %q", i, o.route, o.humanOnly, policy[o.route])
		}
	}
	// Every human-only route's fields are a deliberate MCP omission.
	for route, keys := range fields {
		if policy[route] != "humanOnly" {
			continue
		}
		for _, key := range keys {
			if i := findOmission(route, key, "mcp"); i < 0 || !omissions[i].humanOnly {
				t.Errorf("%s is humanOnly in agent_scope.go, but %s is not listed as a humanOnly MCP omission", route, key)
			}
		}
	}
}

// depsOf returns what key hangs on. A key below a value the client forwards
// as raw JSON (key.*, see flatten) hangs on that value: the caller writes it.
func depsOf(sends map[string][]dep, key string) []dep {
	if d := sends[key]; len(d) > 0 {
		return d
	}
	for k := key; strings.Contains(k, "."); {
		k = strings.TrimSuffix(k[:strings.LastIndex(k, ".")], "[]")
		if d := sends[k+".*"]; len(d) > 0 {
			return d
		}
	}
	return nil
}

// findOmission returns the index of the entry that omits route's key on surf.
func findOmission(route, key, surf string) int {
	for i, o := range omissions {
		if o.route == route && (o.key == "*" || o.key == key) && slices.Contains(strings.Split(o.surfaces, ","), surf) {
			return i
		}
	}
	return -1
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
