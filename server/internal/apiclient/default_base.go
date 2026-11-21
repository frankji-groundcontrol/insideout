package apiclient

import "strings"

// PublicBase is the name a person may see. The compiled-in address is
// never printed; callers say "the default service" instead.
func PublicBase(base string) string {
	if base == "" || trimSlash(base) == trimSlash(DefaultBase) {
		return "the default service"
	}
	if host := hostOf(DefaultBase); host != "" && strings.Contains(base, host) {
		return "the default service"
	}
	return base
}

// RedactDefault replaces the compiled-in address inside a sentence.
func RedactDefault(s string) string {
	if host := hostOf(DefaultBase); host != "" {
		s = strings.ReplaceAll(s, DefaultBase, "the default service")
		if origin := originOf(DefaultBase); origin != "" {
			s = strings.ReplaceAll(s, origin, "the default service")
		}
		s = strings.ReplaceAll(s, host, "the default service")
	}
	return s
}

func trimSlash(s string) string { return strings.TrimRight(s, "/") }

func hostOf(raw string) string {
	raw = trimSlash(raw)
	if i := strings.Index(raw, "://"); i >= 0 {
		raw = raw[i+3:]
	}
	if i := strings.IndexAny(raw, "/"); i >= 0 {
		raw = raw[:i]
	}
	return raw
}

func originOf(raw string) string {
	if i := strings.Index(raw, "://"); i >= 0 {
		rest := raw[i+3:]
		if j := strings.IndexAny(rest, "/"); j >= 0 {
			return raw[:i+3+j]
		}
	}
	return ""
}

// DefaultBase is the hosted API. The CLI and the MCP server call it when no
// --api flag, INSIDEOUT_API or saved login names another server, so a finished
// install can reach the service without any environment variables.
//
// It is a var so a release can retarget the shipped binaries at link time
// (a fork or a second deployment needs no source edit):
//
//	go build -ldflags "-X github.com/frankji-groundcontrol/insideout/server/internal/apiclient.DefaultBase=https://api.example.org/api/v1"
//
// -X replaces only a package-level string var initialised with a literal, and
// scripts/default-api.sh reads this line, so keep the declaration on one line.
var DefaultBase = "https://insideout.yalotein.net/api/v1"
