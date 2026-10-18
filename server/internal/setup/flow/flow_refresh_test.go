package flow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// signInAPI answers the credential exchanges the way the real server does
// (internal/api sessionView: the user's fields plus accessToken and
// refreshToken) and every request-an-email route with the uniform {"ok":true}.
func signInAPI(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login", "/auth/login-code/verify", "/auth/magic-link/consume":
			w.Write([]byte(`{"id":"0b6f9a2e-5c1d-4e8a-9d3f-2a7c4b1e8f60","email":"user@example.org","username":"User","accessToken":"acc-1","refreshToken":"ref-1"}`))
		default:
			w.Write([]byte(`{"ok":true}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// Review-4 FULL-02: setup saved only the 15-minute access token, so the saved
// login died after the access TTL with nothing left to renew it. The refresh
// token the server hands out with every sign-in must reach the config file,
// for each of the three sign-in methods.
func TestRunSavesTheRefreshTokenForEverySignInMethod(t *testing.T) {
	for _, method := range []string{AuthPassword, AuthCode, AuthMagic} {
		t.Run(method, func(t *testing.T) {
			api := signInAPI(t)
			pers := &fakePersister{}
			_, err := Run(context.Background(), Inputs{
				Mode: ModeRemote, BaseURL: api.URL, AuthMethod: method,
				Email: "user@example.org", Secret: "secret-value",
			}, Deps{Prober: &fakeProber{}, Client: apiclient.New(api.URL), Persister: pers})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if len(pers.saved) != 1 {
				t.Fatalf("Save calls = %d, want 1", len(pers.saved))
			}
			raw, _ := json.Marshal(pers.saved[0])
			if !strings.Contains(string(raw), `"token":"acc-1"`) || !strings.Contains(string(raw), `"refresh_token":"ref-1"`) {
				t.Fatalf("saved config = %s, want the access token and refresh_token ref-1", raw)
			}
		})
	}
}
