package api

import (
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

func (s *Server) registerMeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/me", s.requireAuth(s.handleGetMe))
	mux.HandleFunc("PATCH /api/v1/me", s.requireAuth(s.handleUpdateMe))
}

// userView is the client-facing shape of a user — never includes
// PasswordHash or the DB-internal Role flag beyond what's needed.
type userView struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Username  string   `json:"username"`
	AvatarURL *string  `json:"avatarUrl"`
	Bio       string   `json:"bio"`
	Keywords  []string `json:"keywords"`
}

func userResponse(u *store.User) userView {
	return userView{
		ID:        u.ID.String(),
		Email:     u.Email,
		Username:  u.Username,
		AvatarURL: u.AvatarURL,
		Bio:       u.Bio,
		Keywords:  u.Keywords,
	}
}

func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil {
		s.log.Error("get user", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, userResponse(user))
}

type updateMeRequest struct {
	Username  string   `json:"username"`
	Bio       string   `json:"bio"`
	Keywords  []string `json:"keywords"`
	AvatarURL *string  `json:"avatarUrl"`
}

func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())

	var req updateMeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || len(req.Username) > 60 {
		httpx.WriteError(w, http.StatusBadRequest, "username is required (max 60 characters)", "", nil)
		return
	}
	if req.Keywords == nil {
		req.Keywords = []string{}
	}

	user, err := s.store.UpdateProfile(r.Context(), userID, req.Username, req.Bio, req.Keywords, req.AvatarURL)
	if err != nil {
		s.log.Error("update profile", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, userResponse(user))
}
