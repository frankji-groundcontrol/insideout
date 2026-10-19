package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
)

// USER-04: the sign-in UI offers emailed links, codes and reset mails, but
// with no INSIDEOUT_MAIL_API_KEY the sender only logs "mail dev mode — not
// sent". The public config endpoint is what the signed-out client reads, so it
// must say whether mail is actually delivered. signupMode comes from the
// database, so this needs a real Postgres like every store-backed test here;
// without DATABASE_URL it skips.
func TestAuthConfigReportsWhetherMailIsDelivered(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	testlock.SignupMode(t, st.Pool) // reads the signup_mode row another package's test flips
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	cases := []struct {
		name string
		cfg  *config.Config
		want bool
	}{
		{"no mail API key is dev mode", &config.Config{}, false},
		{"a mail API key delivers", &config.Config{MailAPIKey: "test-key"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewServer(st, nil, tc.cfg, log, nil, nil)
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/auth/config", nil))
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode %q: %v", w.Body.String(), err)
			}
			switch body["signupMode"] {
			case store.SignupOpen, store.SignupInvitationOnly, store.SignupClosed:
			default:
				t.Fatalf("signupMode = %v, want a signup mode (body %s)", body["signupMode"], w.Body)
			}
			if got := body["mailEnabled"]; got != tc.want {
				t.Fatalf("mailEnabled = %v, want %v (body %s)", got, tc.want, w.Body)
			}
		})
	}
}
