package api

import (
	"errors"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

type memberView struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func (s *Server) handleListMembers(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	wsID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}

	if _, err := s.store.GetMembership(r.Context(), wsID, userID); errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "workspace not found", "", nil)
		return
	} else if err != nil {
		s.log.Error("get membership", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	members, err := s.store.ListMembers(r.Context(), userID, wsID)
	if err != nil {
		s.log.Error("list members", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	views := make([]memberView, len(members))
	for i, m := range members {
		views[i] = memberView{UserID: m.UserID.String(), Username: m.Username, Email: m.Email, Role: m.Role}
	}
	httpx.WriteJSON(w, http.StatusOK, views)
}

type updateMemberRoleRequest struct {
	Role string `json:"role"`
}

func (s *Server) handleUpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	actorID, _ := UserID(r.Context())
	wsID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	targetID, ok := pathUUID(w, r, "userId")
	if !ok {
		return
	}

	var req updateMemberRoleRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	if req.Role != "admin" && req.Role != "member" {
		httpx.WriteError(w, http.StatusBadRequest, "role must be \"admin\" or \"member\"", "", nil)
		return
	}

	err := s.store.UpdateMemberRole(r.Context(), actorID, wsID, targetID, req.Role)
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only a workspace admin can change member roles", "", nil)
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "member not found", "", nil)
		return
	}
	if err != nil {
		s.log.Error("update member role", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	actorID, _ := UserID(r.Context())
	wsID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	targetID, ok := pathUUID(w, r, "userId")
	if !ok {
		return
	}

	err := s.store.RemoveMember(r.Context(), actorID, wsID, targetID)
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only a workspace admin can remove other members", "", nil)
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "member not found", "", nil)
		return
	}
	if err != nil {
		s.log.Error("remove member", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
