package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// rv5-PARA-05: the server says whether an emailed code or link went out
// ("accepted") or was only logged; `login --code` and `login --magic` print
// that word like forgot-password does instead of always claiming it was sent.
func TestEmailedLoginPrintsTheMailStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true,"mail":"logged"}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("INSIDEOUT_API", srv.URL)
	t.Setenv("INSIDEOUT_TOKEN", "")
	for _, flag := range []string{"--code", "--magic"} {
		restore := swapStdin("\n") // no code or link: the command stops after the request
		stderr := captureStderr(t, func() { runClientCommand(words("login", flag, "a@example.com")) })
		restore()
		if want := apiclient.MailStatusLine("logged"); !strings.Contains(stderr, want) {
			t.Errorf("login %s printed %q, want %q", flag, stderr, want)
		}
	}
}
