package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

const (
	magicLinkTTL     = 15 * time.Minute
	passwordResetTTL = time.Hour
	loginCodeTTL     = 10 * time.Minute
)

type emailOnlyRequest struct {
	Email string `json:"email"`
}

// handleMagicLinkRequest emails a one-time sign-in link. The response is a
// uniform 200 whether or not the account exists or the resend cooldown
// blocked the send — the endpoint must not leak account enumeration, and
// a mail-provider failure is logged, not surfaced, for the same reason.
//
// The uniformity covers time as well as the body (rv4-SEC-03): the lookup is
// the only account-dependent work on the request path, and everything after it
// (the token, its row, the mail) runs on the mail queue, so a known address
// answers as fast as an unknown one and a slow provider never holds a request.
func (s *Server) handleMagicLinkRequest(w http.ResponseWriter, r *http.Request) {
	email, ok := s.decodeEmail(w, r)
	if !ok {
		return
	}
	if !s.authCooldown.allow(email, s.clientIP(r)) {
		// Cooldown hit: same uniform 200 as every other outcome, so the
		// gate never leaks whether the account exists or a mail went out.
		httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	user, err := s.store.GetUserByEmail(r.Context(), email)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	if err != nil {
		s.log.Error("magic link: get user by email", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	s.mail.Enqueue("magic link", func(ctx context.Context) error {
		return s.sendEmailLink(ctx, user.ID, user.Email, store.AuthTokenMagicLink, magicLinkTTL, "/magic-link",
			"Your InsideOut sign-in link",
			"Sign in to InsideOut with this link (valid for 15 minutes):",
			"Sign in to InsideOut",
		)
	})
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleMagicLinkConsume exchanges an emailed token for a full session —
// the same body and cookies as login.
func (s *Server) handleMagicLinkConsume(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil || strings.TrimSpace(req.Token) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "token is required", "", nil)
		return
	}

	userID, err := s.store.ConsumeAuthToken(r.Context(), store.AuthTokenMagicLink, auth.HashOneTimeToken(strings.TrimSpace(req.Token)))
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid or expired link", "", nil)
		return
	}
	if err != nil {
		s.log.Error("magic link: consume", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	user, err := s.store.GetUserByID(r.Context(), userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid or expired link", "", nil)
		return
	}
	if err != nil {
		s.log.Error("magic link: get user", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	access, refresh, err := s.issueSession(w, r, user.ID)
	if err != nil {
		s.log.Error("magic link: issue session", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, sessionView{userView: userResponse(user), AccessToken: access, RefreshToken: refresh})
}

// handleForgotPassword emails a one-time password-reset link. Same
// uniform-200 contract as the magic-link request, including on cooldown and
// in how long the answer takes.
func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	email, ok := s.decodeEmail(w, r)
	if !ok {
		return
	}
	if !s.authCooldown.allow(email, s.clientIP(r)) {
		httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	user, err := s.store.GetUserByEmail(r.Context(), email)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	if err != nil {
		s.log.Error("forgot password: get user by email", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	s.mail.Enqueue("forgot password", func(ctx context.Context) error {
		return s.sendEmailLink(ctx, user.ID, user.Email, store.AuthTokenPasswordReset, passwordResetTTL, "/reset-password",
			"Reset your InsideOut password",
			"Set a new InsideOut password with this link (valid for 60 minutes):",
			"Reset your InsideOut password",
		)
	})
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleResetPassword consumes a reset token, rewrites the password hash,
// and revokes every live session so stolen cookies die with the old
// password.
func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" {
		httpx.WriteError(w, http.StatusBadRequest, "token is required", "", nil)
		return
	}
	if len(req.Password) < 8 {
		httpx.WriteError(w, http.StatusBadRequest, "password must be at least 8 characters", "", nil)
		return
	}

	userID, err := s.store.ConsumeAuthToken(r.Context(), store.AuthTokenPasswordReset, auth.HashOneTimeToken(req.Token))
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid or expired reset link", "", nil)
		return
	}
	if err != nil {
		s.log.Error("reset password: consume", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.log.Error("reset password: hash", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	if err := s.store.UpdatePasswordHash(r.Context(), userID, hash); err != nil {
		s.log.Error("reset password: update hash", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	if err := s.store.RevokeUserSessions(r.Context(), userID); err != nil {
		// The password is already changed; a failed revoke is a security
		// follow-up, not a reason to fail the request (the token is spent
		// either way, so a retry could only 401).
		s.log.Error("reset password: revoke sessions", "error", err, "user_id", userID)
	}

	s.clearAuthCookies(w)
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleLoginCodeRequest emails a 6-digit sign-in code. Same uniform-200
// no-enumeration contract as the magic-link request, including on cooldown and
// in how long the answer takes.
func (s *Server) handleLoginCodeRequest(w http.ResponseWriter, r *http.Request) {
	email, ok := s.decodeEmail(w, r)
	if !ok {
		return
	}
	if !s.authCooldown.allow(email, s.clientIP(r)) {
		httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	user, err := s.store.GetUserByEmail(r.Context(), email)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	if err != nil {
		s.log.Error("login code: get user by email", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	s.mail.Enqueue("login code", func(ctx context.Context) error {
		return s.sendEmailCode(ctx, user.ID, user.Email)
	})
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleLoginCodeVerify exchanges an emailed code for a full session.
// Wrong, exhausted, expired, and unknown-account all answer the same 401 —
// the code itself is the only secret being probed.
func (s *Server) handleLoginCodeVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	code := strings.TrimSpace(req.Code)
	if !emailPattern.MatchString(email) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid email", "", nil)
		return
	}
	if !loginCodePattern.MatchString(code) {
		httpx.WriteError(w, http.StatusBadRequest, "code must be 6 digits", "", nil)
		return
	}

	user, err := s.store.GetUserByEmail(r.Context(), email)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid or expired code", "", nil)
		return
	}
	if err != nil {
		s.log.Error("login code: get user by email", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	if err := s.store.VerifyLoginCode(r.Context(), user.ID, auth.HashOneTimeToken(code)); errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid or expired code", "", nil)
		return
	} else if err != nil {
		s.log.Error("login code: verify", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	access, refresh, err := s.issueSession(w, r, user.ID)
	if err != nil {
		s.log.Error("login code: issue session", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, sessionView{userView: userResponse(user), AccessToken: access, RefreshToken: refresh})
}

func (s *Server) decodeEmail(w http.ResponseWriter, r *http.Request) (string, bool) {
	var req emailOnlyRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return "", false
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if !emailPattern.MatchString(email) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid email", "", nil)
		return "", false
	}
	return email, true
}

// sendEmailLink mints a one-time token of the given kind, stores its hash
// (superseding any prior unused token of that kind for the user), and
// emails a link to the Flutter app's route for it. It runs as a mail-queue
// job, so ctx is the job's, not a request's.
func (s *Server) sendEmailLink(ctx context.Context, userID uuid.UUID, to, kind string, ttl time.Duration, path, subject, textIntro, heading string) error {
	token, hash, err := auth.GenerateOneTimeToken()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}
	if err := s.store.CreateAuthToken(ctx, userID, kind, hash, time.Now().Add(ttl)); err != nil {
		return fmt.Errorf("store token: %w", err)
	}
	link := s.cfg.PublicURL + path + "?token=" + token
	text := fmt.Sprintf("%s\n\n%s\n\nIf you didn't request this, you can ignore this email.", textIntro, link)
	html := simpleEmailHTML(heading, textIntro, link, linkExpiryPhrase(ttl))
	return s.mail.Send(ctx, to, subject, text, html)
}

// sendEmailCode mints a 6-digit sign-in code, stores its hash (superseding
// the user's prior live code), and emails the digits — the typeable
// counterpart of sendEmailLink for clients that can't follow a link.
func (s *Server) sendEmailCode(ctx context.Context, userID uuid.UUID, to string) error {
	code, hash, err := auth.GenerateLoginCode()
	if err != nil {
		return fmt.Errorf("generate code: %w", err)
	}
	if err := s.store.CreateAuthToken(ctx, userID, store.AuthTokenLoginCode, hash, time.Now().Add(loginCodeTTL)); err != nil {
		return fmt.Errorf("store code: %w", err)
	}
	text := fmt.Sprintf("Your InsideOut sign-in code is %s. It works once and expires in %d minutes. If you didn't request it, you can ignore this email.",
		code, int(loginCodeTTL.Minutes()))
	html := codeEmailHTML(code, int(loginCodeTTL.Minutes()))
	return s.mail.Send(ctx, to, "Your InsideOut sign-in code", text, html)
}
