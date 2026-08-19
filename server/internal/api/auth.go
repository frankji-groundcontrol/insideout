package api

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

func (s *Server) registerAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", s.handleRefresh)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
}

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Username = strings.TrimSpace(req.Username)

	if !emailPattern.MatchString(req.Email) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid email", "", nil)
		return
	}
	if len(req.Password) < 8 {
		httpx.WriteError(w, http.StatusBadRequest, "password must be at least 8 characters", "", nil)
		return
	}
	if req.Username == "" || len(req.Username) > 60 {
		httpx.WriteError(w, http.StatusBadRequest, "username is required (max 60 characters)", "", nil)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.log.Error("hash password", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	user, err := s.store.CreateUser(r.Context(), req.Email, hash, req.Username)
	if errors.Is(err, store.ErrConflict) {
		httpx.WriteError(w, http.StatusConflict, "an account with this email already exists", "", nil)
		return
	}
	if err != nil {
		s.log.Error("create user", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	access, refresh, err := s.issueSession(w, r, user.ID)
	if err != nil {
		s.log.Error("issue session", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, sessionView{userView: userResponse(user), AccessToken: access, RefreshToken: refresh})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))

	user, err := s.store.GetUserByEmail(r.Context(), email)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid email or password", "", nil)
		return
	}
	if err != nil {
		s.log.Error("get user by email", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	ok, err := auth.VerifyPassword(user.PasswordHash, req.Password)
	if err != nil || !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid email or password", "", nil)
		return
	}

	// Transparently upgrade a migrated bcrypt hash (from the old
	// juanleme/auth.users table) to argon2id now that we've verified the
	// plaintext password — best-effort, a failure here shouldn't block
	// login.
	if auth.IsBcryptHash(user.PasswordHash) {
		if newHash, err := auth.HashPassword(req.Password); err == nil {
			if err := s.store.UpdatePasswordHash(r.Context(), user.ID, newHash); err != nil {
				s.log.Warn("upgrade bcrypt password hash", "error", err)
			}
		}
	}

	access, refresh, err := s.issueSession(w, r, user.ID)
	if err != nil {
		s.log.Error("issue session", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, sessionView{userView: userResponse(user), AccessToken: access, RefreshToken: refresh})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	raw, err := refreshTokenFromRequest(r)
	if err != nil || raw == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "no refresh token", "", nil)
		return
	}

	hash := auth.HashRefreshToken(raw)
	sess, err := s.store.GetActiveSessionByHash(r.Context(), hash)
	if errors.Is(err, store.ErrNotFound) {
		s.clearAuthCookies(w)
		httpx.WriteError(w, http.StatusUnauthorized, "refresh token expired or already used", "", nil)
		return
	}
	if err != nil {
		s.log.Error("get session", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	newToken, newHash, err := auth.GenerateRefreshToken()
	if err != nil {
		s.log.Error("generate refresh token", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	newExpiry := time.Now().Add(s.cfg.RefreshTTL)
	_, err = s.store.RotateSession(r.Context(), sess.ID, sess.UserID, newHash, newExpiry)
	if errors.Is(err, store.ErrConflict) {
		// A concurrent refresh already rotated this token — this request is a
		// replay. Refuse it and drop the (now-dead) cookie, same contract as
		// the "already used" path above.
		s.clearAuthCookies(w)
		httpx.WriteError(w, http.StatusUnauthorized, "refresh token expired or already used", "", nil)
		return
	}
	if err != nil {
		s.log.Error("rotate session", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	accessToken, err := s.tokens.MintAccessToken(sess.UserID)
	if err != nil {
		s.log.Error("mint access token", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	s.setAuthCookies(w, accessToken, newToken, s.cfg.AccessTTL, s.cfg.RefreshTTL)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "accessToken": accessToken, "refreshToken": newToken})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if raw, err := refreshTokenFromRequest(r); err == nil && raw != "" {
		hash := auth.HashRefreshToken(raw)
		if sess, err := s.store.GetActiveSessionByHash(r.Context(), hash); err == nil {
			_ = s.store.RevokeSession(r.Context(), sess.ID)
		}
	}
	s.clearAuthCookies(w)
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// sessionView is the login/register body: the existing user fields stay
// top-level so Nuxt can keep assigning the JSON to UserProfile, plus the
// tokens Flutter needs when it cannot store httpOnly cookies.
type sessionView struct {
	userView
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// issueSession mints a fresh access+refresh token pair, persists the
// refresh token's hash as a new session row, and sets both cookies.
func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, userID uuid.UUID) (string, string, error) {
	refreshToken, refreshHash, err := auth.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}
	if _, err := s.store.CreateSession(r.Context(), userID, refreshHash, time.Now().Add(s.cfg.RefreshTTL)); err != nil {
		return "", "", err
	}
	accessToken, err := s.tokens.MintAccessToken(userID)
	if err != nil {
		return "", "", err
	}
	s.setAuthCookies(w, accessToken, refreshToken, s.cfg.AccessTTL, s.cfg.RefreshTTL)
	return accessToken, refreshToken, nil
}
