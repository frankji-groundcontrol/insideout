package api

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
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

// storedURLPatch reads a PATCH's avatarUrl or coverUrl (rv5-PARA-09): an
// absent key is nil (keep the stored one), null or "" is "" (clear it), and
// anything else must pass validStoredURL. ok is false once it has answered 400.
func storedURLPatch(w http.ResponseWriter, field string, raw json.RawMessage) (patch *string, ok bool) {
	if raw == nil {
		return nil, true
	}
	var v *string
	if err := json.Unmarshal(raw, &v); err != nil || (v != nil && !validStoredURL(*v)) {
		httpx.WriteError(w, http.StatusBadRequest, field+" must be an https URL (max 2048 characters)", "", nil)
		return nil, false
	}
	if v == nil {
		v = new(string)
	}
	return v, true
}
