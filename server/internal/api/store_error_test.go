package api

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// rv5-STRS-11: one mapping from store errors to status, code and message.
func TestWriteStoreErrorMapsEachStoreErrorOnce(t *testing.T) {
	s := &Server{log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	msgs := errMsgs{store.ErrForbidden: "members only", store.ErrNotFound: "project not found"}
	for _, tc := range []struct {
		err    error
		status int
		code   string
		msg    string
	}{
		{store.ErrNotFound, http.StatusNotFound, "not_found", "project not found"},
		{store.ErrForbidden, http.StatusForbidden, "forbidden", "members only"},
		{store.ErrConflict, http.StatusConflict, "conflict", "conflict"},
		// A wrapped error is matched before the sentinel it wraps.
		{store.ErrHandoverNeedsAdmin, http.StatusForbidden, "handover_needs_admin", ""},
		{store.ErrOwnerNotMember, http.StatusBadRequest, "owner_not_member", ""},
		{store.ErrRepoTaken, http.StatusConflict, "github_repo_already_linked", "this repository is already linked to another project"},
		{fmt.Errorf("%w: a to b", store.ErrInvalidTransition), http.StatusBadRequest, "invalid_transition", "invalid status transition: a to b"},
		{errors.New("boom"), http.StatusInternalServerError, "internal_server_error", "internal error"},
	} {
		rec := httptest.NewRecorder()
		s.writeStoreError(rec, "op", tc.err, msgs)
		got := decodeAs[struct{ Error, Code string }](t, rec.Body.String())
		if rec.Code != tc.status || got.Code != tc.code || (tc.msg != "" && got.Error != tc.msg) {
			t.Errorf("%v: %d %+v, want %d %s %q", tc.err, rec.Code, got, tc.status, tc.code, tc.msg)
		}
	}
}
