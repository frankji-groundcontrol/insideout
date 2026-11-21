package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// answerForgotPassword returns a watch id for every request past the
// cooldown. The id completes only when the emailed link is used. An unknown
// address gets an id that expires pending, so the answer does not reveal
// whether the account exists. A cooled-down request gets no id: the
// cooldown is keyed on the address and the client, not on the account, so
// its absence reveals nothing and the throttled path writes no row.
func (s *Server) answerForgotPassword(w http.ResponseWriter, r *http.Request, email string) {
	if !s.authCooldown.allow(email, s.clientIP(r)) {
		s.writeAuthMail(w, nil)
		return
	}
	expires := s.now().Add(passwordResetTTL)
	var tokenHash string
	{
		user, err := s.store.GetUserByEmail(r.Context(), email)
		switch {
		// Kept on purpose: an unknown account answers like a known one (no enumeration).
		case errors.Is(err, store.ErrNotFound):
		case err != nil:
			s.writeStoreError(w, "forgot password: get user by email", err, nil)
			return
		default:
			token, hash, err := auth.GenerateOneTimeToken()
			if err != nil {
				s.log.Error("forgot password: generate token", "error", err)
				httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
				return
			}
			if err := s.store.CreateAuthToken(r.Context(), user.ID, store.AuthTokenPasswordReset, hash, expires); err != nil {
				s.writeStoreError(w, "forgot password: store token", err, nil)
				return
			}
			tokenHash = hash
			to, raw := user.Email, token
			s.mail.Enqueue("forgot password", func(ctx context.Context) error {
				return s.sendPreparedResetLink(ctx, to, raw)
			})
		}
	}
	watch, err := s.store.CreateResetWatch(r.Context(), tokenHash, expires)
	if err != nil {
		s.writeStoreError(w, "forgot password: create watch", err, nil)
		return
	}
	s.writeAuthMail(w, map[string]any{"watch": watch.String()})
}

func (s *Server) sendPreparedResetLink(ctx context.Context, to, token string) error {
	link := s.cfg.PublicURL + "/reset-password?token=" + token
	textIntro := "Set a new InsideOut password with this link (valid for 60 minutes), or paste the reset code into insideout reset-password. Either works once:"
	text := fmt.Sprintf("%s\n\n%s\n\nReset code:\n%s\n\nIf you didn't request this, you can ignore this email.", textIntro, link, token)
	html := resetEmailHTML(link, token, linkExpiryPhrase(passwordResetTTL))
	return s.mail.Send(ctx, to, "Reset your InsideOut password", text, html)
}

// writeAuthMail answers an emailed-auth request. The mail word is the same
// for a real send and for an unknown address, so the body does not say which
// accounts exist: logged means no provider is configured, accepted means a
// provider is. A send the full queue refused also answers accepted, because
// only a real account reaches the queue; mail.Enqueue logs the drop.
func (s *Server) writeAuthMail(w http.ResponseWriter, extra map[string]any) {
	word := "accepted"
	if s.mail == nil || !s.mail.Enabled() {
		word = "logged"
	}
	body := map[string]any{"ok": true, "mail": word}
	for k, v := range extra {
		body[k] = v
	}
	httpx.WriteJSON(w, http.StatusOK, body)
}

func (s *Server) handleResetWatch(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "expired"})
		return
	}
	status, err := s.store.ResetWatchStatus(r.Context(), id)
	if err != nil {
		s.writeStoreError(w, "reset watch", err, nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": status})
}
