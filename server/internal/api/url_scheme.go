package api

import (
	"net/url"
)

// maxStoredURLLen caps user-supplied display URLs so a hostile client
// cannot park unbounded strings in profile/workspace rows.
const maxStoredURLLen = 2048

// validStoredURL is the door for user-supplied display URLs (avatarUrl,
// coverUrl) that get echoed to other members: empty clears the field,
// otherwise only an absolute https URL passes. javascript:, data: and
// every other scheme — including opaque (https:foo) and protocol-relative
// (//host) forms, which parse without an https host — stay out. The
// codebase has no dev-mode URL pattern, so http gets no exception.
func validStoredURL(raw string) bool {
	if raw == "" {
		return true
	}
	if len(raw) > maxStoredURLLen {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host != ""
}
