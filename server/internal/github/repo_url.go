package github

import (
	"fmt"
	"strings"
)

// ParseRepoURL extracts owner/repo from a github.com URL, a bare "owner/repo",
// with or without scheme, .git suffix, or trailing slash. Anything before
// owner/repo is a host and must be exactly github.com (rv4-SEC-06): after a
// scheme, or when the first segment holds a dot, which a GitHub login never
// does. Segments past owner/repo (a pasted /tree/main) are ignored.
// www.github.com and a login@ before the host are refused on purpose: the SQL
// mirror insideout._canonical_repo_url refuses them too, so widen both (F183).
func ParseRepoURL(raw string) (owner, repo string, err error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimSuffix(s, ".git")
	scheme := false
	if i := strings.Index(s, "://"); i >= 0 {
		s, scheme = s[i+3:], true
	}
	parts := strings.Split(s, "/")
	if scheme || strings.Contains(parts[0], ".") {
		if strings.EqualFold(parts[0], "github.com") {
			parts = parts[1:]
		} else {
			parts = nil // a foreign host: falls through to the error below
		}
	}
	if len(parts) < 2 || !isRepoSegment(parts[0]) || !isRepoSegment(parts[1]) {
		return "", "", fmt.Errorf("github: cannot parse repo %q — want owner/repo or a github.com URL", raw)
	}
	return parts[0], parts[1], nil
}

// CanonicalRepoURL is the one spelling the projects table stores and the
// webhook lookup matches: https://github.com/<owner>/<repo>. The case stays as
// typed, because that lookup is an exact match on GitHub's own full_name.
func CanonicalRepoURL(owner, repo string) string { return RepoURLFor(owner + "/" + repo) }

// isRepoSegment reports whether s is safe to interpolate as a path segment
// of an api.github.com URL: GitHub names are [A-Za-z0-9._-], and anything
// else ('?', '#', '%', spaces…) could reshape the outbound request (SEC-06).
// A segment of only dots (".", "..") is built from allowed characters but
// climbs the request path, so it is refused with the empty segment.
func isRepoSegment(s string) bool {
	if strings.Trim(s, ".") == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}
