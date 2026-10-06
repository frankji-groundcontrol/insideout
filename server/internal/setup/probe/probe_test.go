package probe

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAPIProbe(t *testing.T) {
	cases := []struct {
		name             string
		status           int
		baseURLSuffix    string // "" or "/" to prove the trailing-slash trim
		wantReachable    bool
		wantAuthRequired bool
		wantErr          bool
		wantErrText      string // substring the error must name
	}{
		{"401 is the pass", 401, "", true, true, false, ""},
		{"200 is reachable without auth", 200, "/", true, false, false, ""},
		{"500 names the status", 500, "", true, false, true, "500"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.WriteHeader(tc.status)
			}))
			defer srv.Close()

			res, err := APIProbe(context.Background(), srv.Client(), srv.URL+tc.baseURLSuffix)

			if gotPath != "/me" {
				t.Errorf("probe hit %q, want /me", gotPath)
			}
			if res.Reachable != tc.wantReachable {
				t.Errorf("Reachable = %v, want %v", res.Reachable, tc.wantReachable)
			}
			if res.AuthRequired != tc.wantAuthRequired {
				t.Errorf("AuthRequired = %v, want %v", res.AuthRequired, tc.wantAuthRequired)
			}
			if res.Status != tc.status {
				t.Errorf("Status = %d, want %d", res.Status, tc.status)
			}
			if tc.wantErr {
				if err == nil {
					t.Fatalf("APIProbe = nil error, want one naming %s", tc.wantErrText)
				}
				if !strings.Contains(err.Error(), tc.wantErrText) {
					t.Errorf("APIProbe error %q does not name status %s", err, tc.wantErrText)
				}
				return
			}
			if err != nil {
				t.Fatalf("APIProbe = %v, want nil", err)
			}
		})
	}
}

// Passing a nil client exercises the built-in default; pointing it at a
// server that has been closed gives a refused connection.
func TestAPIProbeUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res, err := APIProbe(context.Background(), nil, url)
	if !errors.Is(err, ErrUnreachable) {
		t.Fatalf("APIProbe = %v, want ErrUnreachable", err)
	}
	if res.Reachable {
		t.Errorf("Reachable = true for a refused connection, want false")
	}
}

func TestGitHubProbe(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		body      string
		wantLogin string
		wantErr   error
	}{
		{"200 decodes the login", 200, `{"login":"octocat"}`, "octocat", nil},
		{"401 is a bad token", 401, `{}`, "", ErrBadToken},
		{"403 is rate limited", 403, `{}`, "", ErrRateLimited},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotAuth string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				w.WriteHeader(tc.status)
				fmt.Fprintln(w, tc.body)
			}))
			defer srv.Close()
			oldURL := githubUserURL
			githubUserURL = srv.URL
			t.Cleanup(func() { githubUserURL = oldURL })

			login, err := GitHubProbe(context.Background(), srv.Client(), "fake-token-for-test")

			if gotAuth != "Bearer fake-token-for-test" {
				t.Errorf("Authorization header = %q, want the Bearer token", gotAuth)
			}
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("GitHubProbe = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GitHubProbe = %v, want nil", err)
			}
			if login != tc.wantLogin {
				t.Errorf("login = %q, want %q", login, tc.wantLogin)
			}
		})
	}
}

func TestGitHubProbeUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	oldURL := githubUserURL
	githubUserURL = srv.URL
	t.Cleanup(func() { githubUserURL = oldURL })

	login, err := GitHubProbe(context.Background(), srv.Client(), "fake-token-for-test")
	if err == nil {
		t.Fatal("GitHubProbe = nil error for 502, want an unexpected-status error")
	}
	for _, sentinel := range []error{ErrBadToken, ErrRateLimited, ErrUnreachable} {
		if errors.Is(err, sentinel) {
			t.Errorf("GitHubProbe error %v must not match %v for an unexpected status", err, sentinel)
		}
	}
	if login != "" {
		t.Errorf("login = %q, want empty on error", login)
	}
}

func TestGitHubProbeUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()
	oldURL := githubUserURL
	githubUserURL = url
	t.Cleanup(func() { githubUserURL = oldURL })

	login, err := GitHubProbe(context.Background(), nil, "fake-token-for-test")
	if !errors.Is(err, ErrUnreachable) {
		t.Fatalf("GitHubProbe = %v, want ErrUnreachable", err)
	}
	if login != "" {
		t.Errorf("login = %q, want empty on error", login)
	}
}

// refusedPortDSN builds a DSN for a port that was just closed, so a connect
// attempt fails fast with connection refused. The userinfo is fake but shaped
// like the real thing so leak assertions are meaningful.
func refusedPortDSN(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve a port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		t.Fatalf("close reserved port: %v", err)
	}
	return fmt.Sprintf("postgres://probe_user:probe_secret@127.0.0.1:%d/probe_db", port)
}

func TestDBProbeRefusedPort(t *testing.T) {
	dsn := refusedPortDSN(t) + "?sslmode=disable&pgbouncer=true"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := DBProbe(ctx, dsn)
	if err == nil {
		t.Fatal("DBProbe = nil error for a refused port, want an error")
	}
	if !errors.Is(err, ErrUnreachable) {
		t.Errorf("DBProbe = %v, want it to wrap ErrUnreachable", err)
	}
	if strings.Contains(err.Error(), dsn) {
		t.Errorf("DBProbe error text leaks the DSN: %v", err)
	}
	if strings.Contains(err.Error(), "probe_secret") {
		t.Errorf("DBProbe error text leaks the password: %v", err)
	}
	if strings.Contains(err.Error(), "pgbouncer") {
		t.Errorf("DBProbe error suggests the pgbouncer param reached pgx: %v", err)
	}

	t.Run("empty DSN", func(t *testing.T) {
		if err := DBProbe(ctx, ""); err == nil {
			t.Error("DBProbe(\"\") = nil, want an error")
		}
	})
}

func TestScrubDSN(t *testing.T) {
	dsn := "postgres://probe_user:probe_secret@h:5432/db?pgbouncer=true"
	cause := fmt.Errorf("failed to connect to %s: dial refused", dsn)

	scrubbed := scrubDSN(cause, dsn, stripPgbouncer(dsn))
	if strings.Contains(scrubbed.Error(), dsn) {
		t.Errorf("scrubbed text still contains the DSN: %v", scrubbed)
	}
	if strings.Contains(scrubbed.Error(), "probe_secret") {
		t.Errorf("scrubbed text still contains the password: %v", scrubbed)
	}
	if !strings.Contains(scrubbed.Error(), "<dsn>") {
		t.Errorf("scrubbed text lost the diagnosis: %v", scrubbed)
	}
	if !errors.Is(scrubbed, cause) {
		t.Error("scrubbing must keep the cause chain usable with errors.Is")
	}
}

// The transform must mirror the sed pipeline in scripts/env.sh: the pgbouncer
// query parameter disappears no matter its position, every other parameter and
// the userinfo survive, and a keyword/value DSN passes through untouched.
func TestStripPgbouncer(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"only param", "postgres://u:p@h:5432/db?pgbouncer=true", "postgres://u:p@h:5432/db"},
		{"first param", "postgres://u:p@h:5432/db?pgbouncer=true&sslmode=disable", "postgres://u:p@h:5432/db?sslmode=disable"},
		{"last param", "postgres://u:p@h:5432/db?sslmode=disable&pgbouncer=true", "postgres://u:p@h:5432/db?sslmode=disable"},
		{"middle param", "postgres://u:p@h:5432/db?sslmode=disable&pgbouncer=true&connect_timeout=5", "postgres://u:p@h:5432/db?sslmode=disable&connect_timeout=5"},
		{"no such param", "postgres://u:p@h:5432/db?sslmode=disable", "postgres://u:p@h:5432/db?sslmode=disable"},
		{"no query at all", "postgres://u:p@h:5432/db", "postgres://u:p@h:5432/db"},
		{"keyword DSN untouched", "host=h password=p sslmode=disable", "host=h password=p sslmode=disable"},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := stripPgbouncer(tc.in); got != tc.want {
				t.Errorf("stripPgbouncer(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestLLMShape(t *testing.T) {
	cases := []struct {
		name    string
		key     string
		wantErr error
	}{
		{"empty is the offline coach mode", "", ErrLLMNotSet},
		{"whitespace only", "   \t\n", ErrLLMBadShape},
		{"short key", "abc123", ErrLLMBadShape},
		{"15 chars is too short", "012345678901234", ErrLLMBadShape},
		{"16 chars is enough", "0123456789012345", nil},
		{"realistic length", "fake-key-0123456789abcdef", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := LLMShape(tc.key)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("LLMShape(%q) = %v, want nil", tc.key, err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("LLMShape(%q) = %v, want %v", tc.key, err, tc.wantErr)
			}
		})
	}
}

func TestDefaultClient(t *testing.T) {
	if got := defaultClient().Timeout; got != 10*time.Second {
		t.Errorf("default client timeout = %v, want 10s", got)
	}
}
