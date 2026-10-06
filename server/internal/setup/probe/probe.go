// Package probe holds the health checks the setup TUI runs at each step:
// API reachability (a 401 on /me is the pass), PostgreSQL reachability via
// pgx, GitHub token validity, and an LLM key shape check. Probes are native,
// not shelled (docs/plans/2026-10-07-setup-tui.md): a live LLM probe costs
// money, so LLMShape only validates shape and a live check stays opt-in.
package probe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Sentinels — callers branch with errors.Is.
var (
	// ErrUnreachable wraps every transport-level failure (dial, DNS, TLS).
	ErrUnreachable = errors.New("unreachable")
	// ErrBadToken means GitHub answered 401 for the token.
	ErrBadToken = errors.New("github token rejected")
	// ErrRateLimited means GitHub answered 403.
	ErrRateLimited = errors.New("github rate limited")
	// ErrLLMNotSet is not a failure: an empty key selects the offline
	// template-reply coach mode (scripts/env.sh reports the same as info).
	ErrLLMNotSet = errors.New("LLM API key not set (offline coach mode)")
	// ErrLLMBadShape means a key is set but cannot plausibly be one.
	ErrLLMBadShape = errors.New("LLM API key malformed")
)

const (
	defaultTimeout = 10 * time.Second
	// minLLMKeyLen is the shortest prefix we accept as plausibly a real key;
	// shape only, no network call.
	minLLMKeyLen = 16
)

// githubUserURL is a var so tests can point the probe at an httptest server.
var githubUserURL = "https://api.github.com/user"

// defaultClient is used whenever a probe is called with a nil client.
func defaultClient() *http.Client { return &http.Client{Timeout: defaultTimeout} }

// APIResult reports what APIProbe learned about the API base URL.
type APIResult struct {
	Reachable bool
	Status    int
	// AuthRequired is true when the API answered 401 — that is the pass:
	// reachable and correctly demanding authentication.
	AuthRequired bool
}

// APIProbe GETs <baseURL>/me (baseURL already includes the path, e.g.
// .../api/v1). A 401 passes with AuthRequired set; any 2xx passes without;
// any other status is reachable but wrong and the error names the status;
// transport failures wrap ErrUnreachable.
func APIProbe(ctx context.Context, client *http.Client, baseURL string) (APIResult, error) {
	if client == nil {
		client = defaultClient()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/me", nil)
	if err != nil {
		return APIResult{}, fmt.Errorf("APIProbe: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return APIResult{}, fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return APIResult{Reachable: true, Status: resp.StatusCode, AuthRequired: true}, nil
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return APIResult{Reachable: true, Status: resp.StatusCode}, nil
	default:
		return APIResult{Reachable: true, Status: resp.StatusCode},
			fmt.Errorf("unexpected status %d from API", resp.StatusCode)
	}
}

// GitHubProbe validates a token against the GitHub user endpoint and returns
// the login it belongs to. 401 wraps ErrBadToken, 403 wraps ErrRateLimited,
// any other status is unexpected, transport failures wrap ErrUnreachable.
func GitHubProbe(ctx context.Context, client *http.Client, token string) (string, error) {
	if client == nil {
		client = defaultClient()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubUserURL, nil)
	if err != nil {
		return "", fmt.Errorf("GitHubProbe: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		var user struct {
			Login string `json:"login"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
			return "", fmt.Errorf("GitHubProbe: decode user: %w", err)
		}
		return user.Login, nil
	case http.StatusUnauthorized:
		return "", ErrBadToken
	case http.StatusForbidden:
		return "", ErrRateLimited
	default:
		return "", fmt.Errorf("unexpected status %d from GitHub", resp.StatusCode)
	}
}

// DBProbe connects to the DSN with pgx and pings it. The pgbouncer query
// parameter is stripped first, exactly like scripts/env.sh does before any
// external probe, because pgx must not receive it.
//
// No returned error may contain the DSN: pgx failures are scrubbed through
// scrubDSN before they leave this function.
func DBProbe(ctx context.Context, dsn string) error {
	if strings.TrimSpace(dsn) == "" {
		return errors.New("DBProbe: empty DSN")
	}
	stripped := stripPgbouncer(dsn)
	conn, err := pgx.Connect(ctx, stripped)
	if err != nil {
		return scrubDSN(fmt.Errorf("%w: %w", ErrUnreachable, err), dsn, stripped)
	}
	defer conn.Close(ctx)
	if err := conn.Ping(ctx); err != nil {
		return scrubDSN(fmt.Errorf("%w: %w", ErrUnreachable, err), dsn, stripped)
	}
	return nil
}

// LLMShape checks the key's shape only: empty selects the offline coach mode
// (ErrLLMNotSet — legal, not a failure); whitespace-only or implausibly short
// keys give ErrLLMBadShape. A live probe would cost money and stays opt-in.
func LLMShape(key string) error {
	switch {
	case key == "":
		return ErrLLMNotSet
	case strings.TrimSpace(key) == "" || len(key) < minLLMKeyLen:
		return ErrLLMBadShape
	default:
		return nil
	}
}

// stripPgbouncer removes the pgx-only `pgbouncer` query parameter from a DSN
// with the same textual transform as the sed pipeline in scripts/env.sh (the
// `--db` probe block): the parameter disappears wherever it sits, every other
// parameter and the userinfo survive, and a keyword/value DSN passes through
// untouched. Userinfo is deliberately KEPT here: env.sh strips it only
// because pg_isready takes the DSN as argv and "argv is not private —
// /proc/<pid>/cmdline is world-readable by default" (scripts/env.sh
// `check --db` comment). This probe connects in-process and never execs, so
// the DSN never enters the process table and nothing needs stripping.
var (
	pgbouncerFirst  = regexp.MustCompile(`\?pgbouncer=[^&]*&`)
	pgbouncerMiddle = regexp.MustCompile(`&pgbouncer=[^&]*&`)
	pgbouncerLast   = regexp.MustCompile(`[?&]pgbouncer=[^&]*$`)
	loneQueryMark   = regexp.MustCompile(`\?$`)
)

func stripPgbouncer(dsn string) string {
	out := pgbouncerFirst.ReplaceAllString(dsn, "?")
	out = pgbouncerMiddle.ReplaceAllString(out, "&")
	out = pgbouncerLast.ReplaceAllString(out, "")
	return loneQueryMark.ReplaceAllString(out, "")
}

// dsnScrubError presents scrubbed text while keeping the cause chain intact
// for errors.Is.
type dsnScrubError struct {
	cause error
	text  string
}

func (e *dsnScrubError) Error() string { return e.text }
func (e *dsnScrubError) Unwrap() error { return e.cause }

// scrubDSN rewrites err's presentation text so no DSN substring survives:
// each named secret string becomes <dsn>, and a URL-form password becomes
// <password>. It is belt-and-braces — pgx's own errors usually redact — but
// the guarantee must not depend on pgx's error format.
func scrubDSN(err error, secrets ...string) error {
	if err == nil {
		return nil
	}
	text := err.Error()
	for _, s := range secrets {
		if s != "" {
			text = strings.ReplaceAll(text, s, "<dsn>")
		}
	}
	if pw := dsnPassword(secrets); pw != "" {
		text = strings.ReplaceAll(text, pw, "<password>")
	}
	return &dsnScrubError{cause: err, text: text}
}

// dsnPassword extracts the URL-decoded password from the first secret that
// parses as a URL with userinfo; "" when there is none.
func dsnPassword(secrets []string) string {
	for _, s := range secrets {
		u, err := url.Parse(s)
		if err != nil || u.User == nil {
			continue
		}
		if pw, ok := u.User.Password(); ok && pw != "" {
			return pw
		}
	}
	return ""
}
