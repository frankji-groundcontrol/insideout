package clientconfig

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Review-4 USER-04: the saved session renews itself on 401, and the rotated
// pair lands back in the file it came from — and only there.

// renewAPI accepts only "Bearer new" and swaps the pair ref-1 for new/ref-2.
func renewAPI(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch {
		case r.URL.Path == "/auth/refresh" && strings.Contains(string(body), `"ref-1"`):
			w.Write([]byte(`{"ok":true,"accessToken":"new","refreshToken":"ref-2"}`))
		case r.Header.Get("Authorization") == "Bearer new":
			w.Write([]byte(`{"email":"a@example.org"}`))
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func pathOf(t *testing.T) string {
	t.Helper()
	p, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	return p
}

func TestResolveHandsOutTheRefreshTokenOnlyWithTheSavedToken(t *testing.T) {
	pinConfig(t, Config{APIBase: "https://a.example/api/v1", Token: "tok-a", RefreshToken: "ref-a"})

	if got := mustResolve(t, ""); got.Token != "tok-a" || got.Refresh != "ref-a" {
		t.Errorf("saved login = %+v, want its token and refresh token", got)
	}
	if got := mustResolve(t, "https://b.example/api/v1"); got.Token != "" || got.Refresh != "" {
		t.Errorf("another base = %+v, want neither token (A's session must not reach B)", got)
	}
	t.Setenv("INSIDEOUT_TOKEN", "env-tok")
	if got := mustResolve(t, ""); got.Token != "env-tok" || got.Refresh != "" {
		t.Errorf("env token = %+v, want it used as given and never renewed", got)
	}
}

func TestSavedSessionRenewsOnceAndStoresTheRotatedPair(t *testing.T) {
	srv := renewAPI(t)
	pinConfig(t, Config{APIBase: srv.URL, Token: "old", RefreshToken: "ref-1"})

	if _, err := mustResolve(t, "").Client().Whoami(); err != nil {
		t.Fatalf("Whoami: %v", err)
	}

	got, err := Load(pathOf(t))
	if err != nil || got != (Config{APIBase: srv.URL, Token: "new", RefreshToken: "ref-2"}) {
		t.Fatalf("stored = %+v (%v), want the rotated pair next to the same api_base", got, err)
	}
	if fi, err := os.Stat(pathOf(t)); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("config mode = %v (%v), want 0600", fi.Mode().Perm(), err)
	}
}

func TestRenewalNeverOverwritesALaterLoginForAnotherServer(t *testing.T) {
	srv := renewAPI(t)
	pinConfig(t, Config{APIBase: srv.URL, Token: "old", RefreshToken: "ref-1"})
	c := mustResolve(t, "").Client()
	other := Config{APIBase: "https://other.example/api/v1", Token: "t2", RefreshToken: "r2"}
	if err := Save(pathOf(t), other); err != nil { // `insideout login --api ...` in another terminal
		t.Fatalf("Save: %v", err)
	}

	if _, err := c.Whoami(); err != nil {
		t.Fatalf("Whoami: %v", err)
	}

	if got, _ := Load(pathOf(t)); got != other {
		t.Errorf("stored = %+v, want the other server's login untouched", got)
	}
}

func TestRenewalThatCannotBeSavedSaysSo(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	srv := renewAPI(t)
	pinConfig(t, Config{APIBase: srv.URL, Token: "old", RefreshToken: "ref-1"})
	c := mustResolve(t, "").Client()
	dir := filepath.Dir(pathOf(t))
	if err := os.Chmod(dir, 0o500); err != nil { // the server retires ref-1; the file cannot take ref-2
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })
	r, w, _ := os.Pipe()
	stderr := os.Stderr
	os.Stderr = w

	_, err := c.Whoami()

	os.Stderr = stderr
	w.Close()
	warning, _ := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Whoami: %v (the renewed session still serves this call)", err)
	}
	if !strings.Contains(string(warning), "could not save") || !strings.Contains(string(warning), "insideout login") {
		t.Errorf("stderr = %q, want a warning to sign in again", warning)
	}
}
