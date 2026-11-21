package api

import (
	"errors"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// errMsgs is what one handler says for each store error it expects. A store
// error it leaves out answers with the default message in storeErrors.
type errMsgs map[error]string

// storeErrors maps the store's errors to a status, a code and a default
// message, once for every handler (rv5-STRS-11). Wrapped errors come before
// the sentinels they wrap, since errors.Is matches both. An empty code is the
// status's own name; an empty message is the error's own text.
var storeErrors = []struct {
	err    error
	status int
	code   string
	msg    string
}{
	{store.ErrHandoverNeedsAdmin, http.StatusForbidden, "handover_needs_admin",
		"only a workspace admin can give the project to someone else or leave it without an owner"},
	{store.ErrOwnerNotMember, http.StatusBadRequest, "owner_not_member", "the new owner must be a member of this workspace"},
	{store.ErrLastAdmin, http.StatusConflict, "last_admin", "a workspace needs at least one admin"},
	{store.ErrInvalidTransition, http.StatusBadRequest, "invalid_transition", ""},
	{store.ErrNotFound, http.StatusNotFound, "", "not found"},
	{store.ErrForbidden, http.StatusForbidden, "", "you may not do this"},
	{store.ErrConflict, http.StatusConflict, "", "conflict"},
	{store.ErrValidation, http.StatusBadRequest, "", "invalid request"},
}

// writeStoreError answers err from a store call: a known store error with its
// status, code and message, anything else with a logged 500 under op.
func (s *Server) writeStoreError(w http.ResponseWriter, op string, err error, msgs errMsgs) {
	for _, e := range storeErrors {
		if !errors.Is(err, e.err) {
			continue
		}
		msg := msgs[e.err]
		if msg == "" {
			msg = e.msg
		}
		if msg == "" {
			msg = err.Error()
		}
		httpx.WriteError(w, e.status, msg, e.code, nil)
		return
	}
	s.log.Error(op, "error", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
}
