package main

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// rv4-FR-F70: login saves a refresh token and nothing in the CLI ends it.
// insideout logout must revoke that session and delete the saved login.

type rvCopyLogoutAPI struct {
	*httptest.Server
	mu     sync.Mutex
	status int
	tokens []string
}

func rvCopyNewLogoutAPI(t *testing.T, status int) *rvCopyLogoutAPI {
	t.Helper()
	a := &rvCopyLogoutAPI{status: status}
	a.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var in struct {
			RefreshToken string `json:"refreshToken"`
		}
		_ = json.Unmarshal(body, &in)
		a.mu.Lock()
		if in.RefreshToken != "" {
			a.tokens = append(a.tokens, in.RefreshToken)
		}
		a.mu.Unlock()
		if a.status != 0 && a.status != http.StatusOK {
			w.WriteHeader(a.status)
			_, _ = w.Write([]byte(`{"error":"unavailable"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(a.Close)
	return a
}

func (a *rvCopyLogoutAPI) refreshTokens() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.tokens...)
}

func rvCopyQuietProxies(t *testing.T) {
	t.Helper()
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		t.Setenv(key, "")
	}
}

func rvCopyRunVerb(t *testing.T, args ...string) (bool, int, string, string) {
	t.Helper()
	var handled bool
	var code int
	stdout, stderr := rvCopyCapture(t, func() {
		handled, code = runClientCommand(args)
	})
	return handled, code, stdout, stderr
}

func TestRvCopyLogoutEndsTheSavedLogin(t *testing.T) {
	api := rvCopyNewLogoutAPI(t, http.StatusOK)
	writeClientConfig(t, clientconfig.Config{APIBase: api.URL, Token: "acc-1", RefreshToken: "ref-1"})
	rvCopyQuietProxies(t)

	handled, code, _, stderr := rvCopyRunVerb(t, "logout")
	if !handled || code != 0 {
		t.Fatalf("logout: handled %v exit %d, want handled and 0\n%s", handled, code, stderr)
	}
	if got := api.refreshTokens(); len(got) != 1 || got[0] != "ref-1" {
		t.Fatalf("server refresh tokens = %v, want one call with the saved refresh token", got)
	}
	if _, err := os.Stat(configPath(t)); !os.IsNotExist(err) {
		t.Fatalf("saved login still present (%v): %+v", err, savedConfig(t))
	}
	listed := false
	for _, v := range cliVerbs {
		if v.name == "logout" {
			listed = true
		}
	}
	if !listed {
		t.Fatal("cliVerbs has no logout")
	}
	var help strings.Builder
	printHelp(&help)
	if !strings.Contains(help.String(), "\n  logout ") {
		t.Fatal("insideout help does not list logout")
	}

	handled, code, _, stderr = rvCopyRunVerb(t, "logout")
	if !handled || code != 0 || !strings.Contains(stderr, "not signed in") {
		t.Fatalf("second logout: handled %v exit %d, stderr %q, want \"not signed in\" and exit 0", handled, code, stderr)
	}
	if got := api.refreshTokens(); len(got) != 1 {
		t.Fatalf("second logout called the server again: %v", got)
	}
}

func TestRvCopyLogoutWithNoSavedLogin(t *testing.T) {
	rvCopyPinHome(t)
	handled, code, _, stderr := rvCopyRunVerb(t, "logout")
	if !handled || code != 0 || strings.TrimSpace(stderr) != "not signed in" {
		t.Fatalf("no saved login: handled %v exit %d, stderr %q, want one line \"not signed in\" and exit 0", handled, code, stderr)
	}
}

func TestRvCopyLogoutKeepsTheLoginWhenTheServerRefuses(t *testing.T) {
	api := rvCopyNewLogoutAPI(t, http.StatusServiceUnavailable)
	kept := clientconfig.Config{APIBase: api.URL, Token: "acc-1", RefreshToken: "ref-1"}
	writeClientConfig(t, kept)
	rvCopyQuietProxies(t)

	handled, code, _, stderr := rvCopyRunVerb(t, "logout")
	if !handled || code == 0 {
		t.Fatalf("refused logout: handled %v exit %d, stderr %q, want a handled failure", handled, code, stderr)
	}
	if got := api.refreshTokens(); len(got) != 1 || got[0] != "ref-1" {
		t.Fatalf("server refresh tokens = %v, want the saved refresh token", got)
	}
	if got := savedConfig(t); got != kept {
		t.Fatalf("saved login = %+v, want it kept for a retry", got)
	}
}

func TestRvCopyLogoutDeletesTheLoginWhenTheServerIsUnreachable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	writeClientConfig(t, clientconfig.Config{APIBase: "http://" + addr, Token: "acc-1", RefreshToken: "ref-1"})
	rvCopyQuietProxies(t)

	handled, code, _, stderr := rvCopyRunVerb(t, "logout")
	if !handled || code != 0 {
		t.Fatalf("unreachable logout: handled %v exit %d, want handled and 0\n%s", handled, code, stderr)
	}
	if !strings.Contains(stderr, "could not be revoked") || !strings.Contains(stderr, "expires by itself") {
		t.Fatalf("stderr %q, want the server session could not be revoked and expires by itself", stderr)
	}
	if _, err := os.Stat(configPath(t)); !os.IsNotExist(err) {
		t.Fatalf("saved login still present (%v)", err)
	}
}

func TestRvCopyLogoutLeavesAnotherServersLogin(t *testing.T) {
	api := rvCopyNewLogoutAPI(t, http.StatusOK)
	kept := clientconfig.Config{APIBase: api.URL, Token: "acc-1", RefreshToken: "ref-1"}
	writeClientConfig(t, kept)
	rvCopyQuietProxies(t)

	handled, code, _, stderr := rvCopyRunVerb(t, "logout", "--api", "http://127.0.0.1:9")
	if !handled || code != 0 || !strings.Contains(stderr, "not signed in") {
		t.Fatalf("logout for another base: handled %v exit %d, stderr %q, want \"not signed in\"", handled, code, stderr)
	}
	if got := savedConfig(t); got != kept {
		t.Fatalf("saved login = %+v, want the other server's login kept", got)
	}
	if got := api.refreshTokens(); len(got) != 0 {
		t.Fatalf("logout for another base called the saved server: %v", got)
	}
}

func TestRvCopyLogoutBlanksOnlyTheTokenFields(t *testing.T) {
	api := rvCopyNewLogoutAPI(t, http.StatusOK)
	writeClientConfig(t, clientconfig.Config{})
	path := configPath(t)
	body := `{"api_base":"` + api.URL + `","token":"acc-1","refresh_token":"ref-1","note":"keep-me"}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	rvCopyQuietProxies(t)

	handled, code, _, stderr := rvCopyRunVerb(t, "logout")
	if !handled || code != 0 {
		t.Fatalf("logout: handled %v exit %d\n%s", handled, code, stderr)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("config file gone (%v); a file that holds more than the login keeps the other fields", err)
	}
	if strings.Contains(string(data), "ref-1") || strings.Contains(string(data), "acc-1") || !strings.Contains(string(data), "keep-me") {
		t.Fatalf("config after logout = %s, want the tokens gone and the other field kept", data)
	}
}

func TestRvCopyResetPasswordDropsTheSavedLogin(t *testing.T) {
	api := rvCopyNewLogoutAPI(t, http.StatusOK)
	writeClientConfig(t, clientconfig.Config{APIBase: api.URL, Token: "acc-1", RefreshToken: "ref-1"})

	res := runCLI(t, cliEnv(), "reset-token\nnew-password\n", "reset-password")
	if res.code != 0 {
		t.Fatalf("reset-password: exit %d\n%s", res.code, res.output())
	}
	if _, err := os.Stat(configPath(t)); !os.IsNotExist(err) {
		t.Fatalf("saved login still present after reset-password (%v)", err)
	}
}

func TestRvCopyLogoutIsDocumentedNextToLogin(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "docs", "usage", "cli-mcp-and-guidance.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "config.json") {
		t.Error("cli-mcp-and-guidance.md does not name the saved login file")
	}
	if !strings.Contains(text, "insideout logout") && !strings.Contains(text, "`logout`") {
		t.Error("cli-mcp-and-guidance.md does not document logout next to login")
	}
}
