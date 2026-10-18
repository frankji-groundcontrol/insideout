package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/google/uuid"
)

// rv4-USER-07: create-invitation used to answer 201 with a one-time code and
// signup link and say nothing about whether the invitation e-mail went out.
// The mail sender logs and swallows a provider failure, and in dev mode
// (no key) it "succeeds" without delivering, so an admin watching the
// dialog could tell the invitee "check your inbox" about a message that
// never left. The response now carries mailSent: true only when the sender
// is enabled and the provider accepted the message.

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// fakeResend answers the mail sender's HTTP call in-process with the given
// status, so no test reaches the real provider. The sender's client has no
// transport of its own, so it follows http.DefaultTransport at request time;
// package api runs its tests sequentially, so the swap is race-free. The
// returned func lists the recipients the provider was asked to mail.
func fakeResend(t *testing.T, status int) (recipients func() []string) {
	t.Helper()
	var to []string
	prev := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var payload struct {
			To string `json:"to"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		to = append(to, payload.To)
		return &http.Response{
			StatusCode: status,
			Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("{}")),
			Request:    r,
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = prev })
	return func() []string { return to }
}

func TestAdminCreateInvitationReportsMailSent(t *testing.T) {
	w, admin := newAdminWorld(t)
	keyed := config.Config{PublicURL: "http://app.example.com", MailAPIKey: "re_not_a_real_key", MailFrom: "InsideOut <noreply@example.com>"}
	cases := []struct {
		name     string
		cfg      config.Config
		provider int // HTTP status the fake provider answers; 0 = unused
		wantSent bool
		wantMail int
	}{
		{"dev mode logs instead of delivering", config.Config{PublicURL: "http://app.example.com"}, 0, false, 0},
		{"provider accepts the message", keyed, http.StatusOK, true, 1},
		{"provider rejects the message", keyed, http.StatusInternalServerError, false, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recipients := func() []string { return nil }
			if tc.provider != 0 {
				recipients = fakeResend(t, tc.provider)
			}
			world := rebuildWithLogger(w, &tc.cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
			email := "invitee-" + uuid.NewString() + "@example.com"

			status, body := world.call(admin, http.MethodPost, "/api/v1/admin/invitations", `{"email":"`+email+`","ttlHours":1}`)
			expect(t, "create invitation", status, body, http.StatusCreated)
			got := decodeAs[struct {
				Code      string
				SignupURL string `json:"signupUrl"`
				MailSent  *bool  `json:"mailSent"`
			}](t, body)

			// The link is the admin's fallback whatever the mail did.
			if got.Code == "" || !strings.HasSuffix(got.SignupURL, "/register?invite="+got.Code) {
				t.Fatalf("response must still carry the one-time code and a signup link carrying it")
			}
			if got.MailSent == nil {
				t.Fatalf("response has no mailSent field; the admin cannot tell whether the invitee was e-mailed")
			}
			if *got.MailSent != tc.wantSent {
				t.Errorf("mailSent = %v, want %v", *got.MailSent, tc.wantSent)
			}
			if mails := recipients(); len(mails) != tc.wantMail || (tc.wantMail == 1 && mails[0] != email) {
				t.Errorf("provider was asked to mail %v, want %d message(s) to the invitee", mails, tc.wantMail)
			}
		})
	}
}
