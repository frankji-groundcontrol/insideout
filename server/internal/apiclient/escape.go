package apiclient

import (
	"net/url"
	"strings"
)

// Every id, code, format and audience reaches the API as a person typed it or
// a model chose it (the MCP tools pass LLM-supplied strings straight in), so
// none is glued into a request path or query by hand (rv4-QUAL-03): each value
// becomes exactly one path segment or one query value, whatever it contains.

// apiPath joins segments into a request path. Each argument is exactly one
// segment: a "/" inside it is escaped, not a separator.
func apiPath(segs ...string) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteByte('/')
		b.WriteString(escSeg(s))
	}
	return b.String()
}

// escSeg is url.PathEscape plus the two dot segments, which PathEscape leaves
// bare and which a server (or a proxy before it) resolves as "." and "..": an
// id of ".." would otherwise climb out of its route.
func escSeg(s string) string {
	switch s {
	case ".":
		return "%2E"
	case "..":
		return "%2E%2E"
	}
	return url.PathEscape(s)
}

// withQuery appends k=v pairs (alternating key, value) to path in the order
// given, query-escaping both. url.Values.Encode would sort the keys and so
// reorder URLs that valid callers already send.
func withQuery(path string, kv ...string) string {
	sep := "?"
	var b strings.Builder
	b.WriteString(path)
	for i := 0; i+1 < len(kv); i += 2 {
		b.WriteString(sep + url.QueryEscape(kv[i]) + "=" + url.QueryEscape(kv[i+1]))
		sep = "&"
	}
	return b.String()
}
