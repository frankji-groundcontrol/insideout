package api

import (
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
)

func (s *Server) registerDeleteMeRoute(mux *http.ServeMux) {
	mux.HandleFunc("DELETE /api/v1/me", s.requireAuth(s.handleDeleteMe))
}

type deleteMeRequest struct {
	Password string `json:"password"`
}

// handleDeleteMe tombstones the signed-in account. The current password is
// required again. The route is human-only (agent_scope.go): an agent token
// cannot erase the account it is acting for.
func (s *Server) handleDeleteMe(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	var req deleteMeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	if req.Password == "" {
		httpx.WriteError(w, http.StatusBadRequest, "current password is required", "", nil)
		return
	}
	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil {
		s.writeStoreError(w, "delete me: get user", err, nil)
		return
	}
	cred, err := s.store.GetUserByEmail(r.Context(), user.Email)
	if err != nil {
		s.writeStoreError(w, "delete me: credential", err, nil)
		return
	}
	ok, err := auth.VerifyPassword(cred.PasswordHash, req.Password)
	if err != nil {
		s.log.Error("delete me: verify password", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	if !ok {
		httpx.WriteError(w, http.StatusForbidden, "current password does not match", "", nil)
		return
	}
	if err := s.store.EraseUser(r.Context(), userID); err != nil {
		s.writeStoreError(w, "delete me", err, nil)
		return
	}
	s.clearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}
