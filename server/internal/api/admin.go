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
// lives in the database.
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
	if err := s.mail.Send(r.Context(), email, subject, text, html); err != nil {
		s.log.Error("admin: send invitation", "error", err)
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"invitation": invitationResponse(inv),
		"code":       code,
		"signupUrl":  link,
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
		simpleEmailHTML("Your invitation", intro, link, int(ttl.Hours()))
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
