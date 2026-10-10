package api

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/google/uuid"
)

// ARCH-03 (review-3): the mail funnel has no delivery signal. The auth
// email flows answer every request with a uniform 200 (correct for
// privacy), mailEnabled only reports key presence, and healthz only pings
// the database — so a hosted deployment whose key is set but whose domain
// is unverified silently delivers nothing, with a log line as the only
// trace. The admin mail probe is that signal: POST a typed recipient, one
// Send, the outcome comes back to the admin. Live-provider behavior is
// owner-blocked and out of scope here; these tests pin the disabled and
// dev-mode paths and recipient validation.

const mailProbePath = "/api/v1/admin/mail/probe"

type probeResponse struct {
	MailEnabled bool   `json:"mailEnabled"`
	Status      string `json:"status"`
	Detail      string `json:"detail"`
	Error       string `json:"error"`
	LastError   string `json:"lastError"`
}

// rebuildWithLogger swaps the world's server for one whose logger the test
// can read, so the dev-mode send the probe triggers is observable.
func rebuildWithLogger(w *routeWorld, cfg *config.Config, log *slog.Logger) *routeWorld {
	srv := NewServer(w.st, w.srv.tokens, cfg, log, nil, nil)
	c := *w
	c.srv = srv
	c.h = srv.Handler()
	return &c
}

// With no key configured the probe must report disabled as a normal 200
// outcome — not an error — and must not claim the message was sent.
func TestAdminMailProbeReportsDisabledWithoutKey(t *testing.T) {
	w, admin := newAdminWorld(t)
	status, body := w.call(admin, http.MethodPost, mailProbePath, `{"email":"probe-`+uuid.NewString()+`@test.local"}`)
	expect(t, "probe without a key", status, body, http.StatusOK)
	res := decodeAs[probeResponse](t, body)
	if res.MailEnabled {
		t.Fatalf("probe without a key: mailEnabled true, want false")
	}
	if res.Status != "disabled" {
		t.Fatalf("probe without a key: status %q, want disabled", res.Status)
	}
	if res.Error != "" {
		t.Fatalf("probe without a key: disabled is a state, not an error; got error %q", res.Error)
	}
}

// With the dev-mode sender the probe reports the dev-mode outcome: the
// Send actually runs (logged, not delivered) and the response says so —
// key-presence sniffing without a send would hide a broken funnel.
func TestAdminMailProbeExercisesDevModeSender(t *testing.T) {
	w, admin := newAdminWorld(t)
	var buf bytes.Buffer
	dev := rebuildWithLogger(w, &config.Config{PublicURL: "http://app.test", MailDevBody: true},
		slog.New(slog.NewTextHandler(&buf, nil)))
	email := "probe-" + uuid.NewString() + "@test.local"

	status, body := dev.call(admin, http.MethodPost, mailProbePath, `{"email":"`+email+`"}`)
	expect(t, "probe in dev mode", status, body, http.StatusOK)
	res := decodeAs[probeResponse](t, body)
	if res.Status != "disabled" {
		t.Fatalf("probe in dev mode: status %q, want disabled", res.Status)
	}
	if !strings.Contains(strings.ToLower(res.Detail), "logged") {
		t.Fatalf("probe in dev mode: detail should say the message was logged, not delivered; got %q", res.Detail)
	}
	if !strings.Contains(buf.String(), "mail dev mode — not sent") || !strings.Contains(buf.String(), email) {
		t.Fatalf("probe must run one real Send through the dev-mode sender; log was %q", buf.String())
	}
}

// A malformed recipient is a 400 before anything is sent.
func TestAdminMailProbeValidatesRecipient(t *testing.T) {
	w, admin := newAdminWorld(t)
	for _, bad := range []string{"not-an-email", ""} {
		status, body := w.call(admin, http.MethodPost, mailProbePath, `{"email":"`+bad+`"}`)
		if status != http.StatusBadRequest {
			t.Fatalf("probe with recipient %q: status %d, want 400; body %s", bad, status, body)
		}
	}
}

// The probe is admin-only, like the rest of the admin surface.
func TestAdminMailProbeAdminOnly(t *testing.T) {
	w, _ := newAdminWorld(t)
	for who, user := range map[string]routeUser{"workspace owner": w.owner, "outsider": w.outsider} {
		status, body := w.call(user, http.MethodPost, mailProbePath, `{"email":"probe-`+uuid.NewString()+`@test.local"}`)
		if status != http.StatusForbidden || !strings.Contains(body, "admin only") {
			t.Errorf("%s: status %d body %s, want 403 admin only", who, status, body)
		}
	}
}
