// Package prdcommit implements the human Commit (PRODUCT.md "Product
// version control"): freeze a working PRD as an immutable version with
// name, audience, summary, carried unknowns, and a section-level diff
// against the previous commit.
package prdcommit

import "sort"

// Diff summarizes what changed between two section maps: per section,
// "added", "removed", or "changed" with the byte delta. Keys are sorted
// so the recorded diff is deterministic.
func Diff(prev, curr map[string]string) map[string]any {
	keys := map[string]bool{}
	for k := range prev {
		keys[k] = true
	}
	for k := range curr {
		keys[k] = true
	}
	names := make([]string, 0, len(keys))
	for k := range keys {
		names = append(names, k)
	}
	sort.Strings(names)

	out := map[string]any{"sections": map[string]any{}}
	added, removed, changed := 0, 0, 0
	sections := out["sections"].(map[string]any)
	for _, k := range names {
		p, hasP := prev[k]
		c, hasC := curr[k]
		switch {
		case !hasP:
			sections[k] = map[string]any{"change": "added", "bytes": len(c)}
			added++
		case !hasC:
			sections[k] = map[string]any{"change": "removed", "bytes": len(p)}
			removed++
		case p != c:
			sections[k] = map[string]any{"change": "changed", "bytes_delta": len(c) - len(p)}
			changed++
		}
	}
	out["counts"] = map[string]int{"added": added, "removed": removed, "changed": changed}
	return out
}
