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

	// Membership pre-check, kept on purpose: a non-member gets 404, not a roster.
	if _, err := s.store.GetMembership(r.Context(), wsID, userID); errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "workspace not found", "", nil)
		return
	} else if err != nil {
		s.writeStoreError(w, "get membership", err, nil)
		return
	}

	members, err := s.store.ListMembers(r.Context(), userID, wsID)
	if errors.Is(err, store.ErrForbidden) {
		// Removed from the workspace between the check above and this read.
		httpx.WriteError(w, http.StatusNotFound, "workspace not found", "", nil)
		return
	}
	if err != nil {
		s.writeStoreError(w, "list members", err, nil)
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
	if err != nil {
		s.writeStoreError(w, "update member role", err, errMsgs{
			store.ErrForbidden: "only a workspace admin can change member roles",
			store.ErrNotFound:  "member not found",
		})
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
	if err != nil {
		s.writeStoreError(w, "remove member", err, errMsgs{
			store.ErrForbidden: "only a workspace admin can remove other members",
			store.ErrNotFound:  "member not found",
		})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
