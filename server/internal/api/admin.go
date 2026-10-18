package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// Admin surface: invitation management and the signup-mode switch. The
// admin gate is users.system_role = 'admin' (set by the promote-admin
// operator subcommand — there is deliberately no self-promotion route).

const defaultInviteTTL = 7 * 24 * time.Hour

func (s *Server) registerAdminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/settings", s.requireAdmin(s.handleGetAdminSettings))
	mux.HandleFunc("PUT /api/v1/admin/settings", s.requireAdmin(s.handlePutAdminSettings))
	mux.HandleFunc("POST /api/v1/admin/invitations", s.requireAdmin(s.handleCreateInvitation))
	mux.HandleFunc("GET /api/v1/admin/invitations", s.requireAdmin(s.handleListInvitations))
	mux.HandleFunc("POST /api/v1/admin/invitations/{id}/revoke", s.requireAdmin(s.handleRevokeInvitation))
	mux.HandleFunc("POST /api/v1/admin/mail/probe", s.requireAdmin(s.handleAdminMailProbe))
}

// requireAdmin chains requireAuth with the system_role check. The 403
// body stays identical for non-admins and unknown users.
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := UserID(r.Context())
		u, err := s.store.GetUserByID(r.Context(), userID)
		if err != nil || u.SystemRole != "admin" {
			httpx.WriteError(w, http.StatusForbidden, "admin only", "", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleGetAdminSettings(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"signupMode": s.store.SignupMode(r.Context())})
}

func (s *Server) handlePutAdminSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SignupMode string `json:"signupMode"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil ||
		req.SignupMode != store.SignupInvitationOnly && req.SignupMode != store.SignupOpen && req.SignupMode != store.SignupClosed {
		httpx.WriteError(w, http.StatusBadRequest, "signupMode must be invitation_only, open, or closed", "", nil)
		return
	}
	if err := s.store.SetSetting(r.Context(), "signup_mode", `"`+req.SignupMode+`"`); err != nil {
		s.log.Error("admin: set signup mode", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"signupMode": req.SignupMode})
}

type invitationView struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	ExpiresAt string  `json:"expiresAt"`
	UsedAt    *string `json:"usedAt"`
	RevokedAt *string `json:"revokedAt"`
	CreatedAt string  `json:"createdAt"`
}

func invitationResponse(i *store.Invitation) invitationView {
	v := invitationView{
		ID:        i.ID.String(),
		Email:     i.Email,
		ExpiresAt: i.ExpiresAt.Format(time.RFC3339),
		UsedAt:    nil,
		RevokedAt: nil,
		CreatedAt: i.CreatedAt.Format(time.RFC3339),
	}
	if i.UsedAt != nil {
		t := i.UsedAt.Format(time.RFC3339)
		v.UsedAt = &t
	}
	if i.RevokedAt != nil {
		t := i.RevokedAt.Format(time.RFC3339)
		v.RevokedAt = &t
	}
	return v
}

// handleCreateInvitation mints an email-bound invitation code and emails
// it to the invitee. The code is returned exactly once — only its sha256
// lives in the database. The mail is best-effort: a send failure is logged
// and the invitation still stands, so the response says whether the mail
// actually went out (mailSent: the sender is enabled — dev mode only logs —
// and the provider accepted the message). When it is false the admin hands
// over signupUrl themselves; the CLI and MCP pass the field through.
func (s *Server) handleCreateInvitation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		TTLHours int    `json:"ttlHours"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if !emailPattern.MatchString(email) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid email", "", nil)
		return
	}
	ttl := time.Duration(req.TTLHours) * time.Hour
	if ttl <= 0 || ttl > 30*24*time.Hour {
		ttl = defaultInviteTTL
	}

	code, hash, err := auth.GenerateOneTimeToken()
	if err != nil {
		s.log.Error("admin: generate invitation", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	adminID, _ := UserID(r.Context())
	inv, err := s.store.CreateInvitation(r.Context(), adminID, email, hash, time.Now().Add(ttl))
	if err != nil {
		s.log.Error("admin: create invitation", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	link := s.cfg.PublicURL + "/register?invite=" + code
	subject, text, html := invitationEmail(link, ttl)
	sendErr := s.mail.Send(r.Context(), email, subject, text, html)
	if sendErr != nil {
		s.log.Error("admin: send invitation", "error", sendErr)
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"invitation": invitationResponse(inv),
		"code":       code,
		"signupUrl":  link,
		"mailSent":   s.mail.Enabled() && sendErr == nil,
	})
}

func humanTTL(ttl time.Duration) string {
	h := int(ttl.Hours())
	if h%24 == 0 {
		return fmt.Sprintf("in %d day(s)", h/24)
	}
	return fmt.Sprintf("in %d hour(s)", h)
}

// invitationEmail builds the invitation mail's subject and both bodies in
// one place so tests can pin them against each other — the HTML footer
// once misstated a 7-day link as "168 minutes" (review-3 FULL-01) because
// the two bodies were assembled inline at the send call.
func invitationEmail(link string, ttl time.Duration) (subject, text, html string) {
	intro := "You are invited to InsideOut. Open this link to create your account (the invitation works once and expires " + humanTTL(ttl) + "):"
	return "Your InsideOut invitation",
		intro + "\n\n" + link + "\n\nIf you weren't expecting this, you can ignore this email.",
		simpleEmailHTML("Your invitation", intro, link, linkExpiryPhrase(ttl))
}

func (s *Server) handleListInvitations(w http.ResponseWriter, r *http.Request) {
	invs, err := s.store.ListInvitations(r.Context())
	if err != nil {
		s.log.Error("admin: list invitations", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	out := make([]invitationView, 0, len(invs))
	for _, i := range invs {
		out = append(out, invitationResponse(i))
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (s *Server) handleRevokeInvitation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid invitation id", "", nil)
		return
	}
	if err := s.store.RevokeInvitation(r.Context(), id); err == store.ErrNotFound {
		httpx.WriteError(w, http.StatusNotFound, "invitation not found or already used", "", nil)
		return
	} else if err != nil {
		s.log.Error("admin: revoke invitation", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleAdminMailProbe is the mail funnel's working-signal (review-3
// ARCH-03). The auth email flows answer every request with a uniform 200 —
// deliberately, so callers can't probe addresses — and mailEnabled only
// reports key presence, so a deployment whose key is set but whose domain
// is unverified silently delivers nothing, with a log line as the only
// trace. The probe sends one typed test message and reports the provider's
// verdict to the admin instead.
//
// Operator note: with the default From (InsideOut <onboarding@resend.dev>)
// Resend delivers only to the account owner's own verified address — a
// probe to any other recipient comes back with the provider's refusal.
// Verify a domain (INSIDEOUT_MAIL_FROM), then exercise one real invitee
// before opening signup.
func (s *Server) handleAdminMailProbe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if !emailPattern.MatchString(email) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid email", "", nil)
		return
	}

	// Snapshot the last failure of any earlier send (a user's magic link,
	// an invitation) before this probe's own send overwrites the record.
	lastError := s.mail.LastSendError()

	// The send always runs: in dev mode it logs the message and returns
	// nil, so the probe exercises the same path every other send takes.
	sendErr := s.mail.Send(r.Context(), email,
		"InsideOut mail probe",
		"This is a test send from the InsideOut admin mail probe. If you received it, delivery to this address works.",
		"")

	res := map[string]any{
		"mailEnabled": s.mail.Enabled(),
		"lastError":   lastError,
	}
	switch {
	case !s.mail.Enabled():
		res["status"] = "disabled"
		res["detail"] = "no mail API key configured; the message was logged (dev mode), not delivered"
	case sendErr != nil:
		res["status"] = "error"
		res["error"] = sendErr.Error()
	default:
		res["status"] = "sent"
	}
	s.log.Info("admin: mail probe", "to", email, "status", res["status"], "error", sendErr)
	httpx.WriteJSON(w, http.StatusOK, res)
}
