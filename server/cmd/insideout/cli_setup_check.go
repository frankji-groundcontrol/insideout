package main

// `insideout setup --check` (slice 3): re-run the health probes
// against the CURRENT configuration — API base + credential from
// clientconfig.Resolve, the database and GitHub/LLM keys from the
// repo .env when a checkout is present. With a credential (the saved
// login or INSIDEOUT_TOKEN) the API is asked as that user, renewing
// an expired saved login like every verb; with none it is the
// anonymous probe, where a 401 is the pass.
// Offline-safe: a probe whose target is unreachable reports SKIP
// with a message instead of failing the run; a configured target
// that answers wrongly (a rejected token, a 5xx) fails it.
// The probes run under the CLI context (cliCtx), like every other verb:
// Ctrl-C cancels them, a probe it cuts short reports SKIP "cancelled",
// and the run ends with exit 1 (runInterruptible turns that into 130),
// never with "all configured checks passed".

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
	"github.com/frankji-groundcontrol/insideout/server/internal/setup/probe"
)

// checkOutcome is one probe line: status is ok | fail | skip.
type checkOutcome struct {
	name   string
	status string
	detail string
}

// commandInterrupted reports that ctx was cancelled, which only the CLI context
// does (Ctrl-C or SIGTERM); the check's own 60 s deadline is DeadlineExceeded.
func commandInterrupted(ctx context.Context) bool {
	return errors.Is(ctx.Err(), context.Canceled)
}

// cancelledOutcome is the line of a probe the interrupt cut short. It is a
// skip, not a failure: the target was never judged.
func cancelledOutcome(name string) checkOutcome {
	return checkOutcome{name, "skip", "cancelled (interrupted before it answered)"}
}

// runHealthCheck is `insideout setup --check`: one line per probe.
func runHealthCheck(llmLive bool) (bool, int) {
	ctx, cancel := context.WithTimeout(cliCtx, 60*time.Second)
	defer cancel()

	outcomes := []checkOutcome{checkAPI(ctx)}
	outcomes = append(outcomes, checkDB(ctx)...)
	outcomes = append(outcomes, checkGitHub(ctx)...)
	outcomes = append(outcomes, checkLLM(ctx, llmLive)...)

	failed := 0
	fmt.Println("InsideOut setup — health check")
	for _, o := range outcomes {
		mark := map[string]string{"ok": "✓", "fail": "✗", "skip": "·"}[o.status]
		fmt.Printf("  %s %-8s %s — %s\n", mark, o.status, o.name, o.detail)
		if o.status == "fail" {
			failed++
		}
	}
	if failed > 0 {
		fmt.Printf("%d check(s) failed\n", failed)
		return true, 1
	}
	if commandInterrupted(ctx) {
		fmt.Println("interrupted — the run did not finish, so it is not a pass")
		return true, 1
	}
	fmt.Println("all configured checks passed (skipped ones need their keys set)")
	return true, 0
}

func checkAPI(ctx context.Context) checkOutcome {
	creds, err := clientconfig.Resolve("")
	if err != nil {
		return checkOutcome{"api", "fail", err.Error()}
	}
	base := creds.Base
	if creds.Token != "" {
		return checkSignedIn(ctx, creds)
	}
	res, err := probe.APIProbe(ctx, nil, base)
	switch {
	case err == nil && res.AuthRequired:
		return checkOutcome{"api", "ok", fmt.Sprintf("%s reachable, authentication required (as expected)", redactBase(base))}
	case err == nil:
		return checkOutcome{"api", "ok", fmt.Sprintf("%s reachable and already authenticated", redactBase(base))}
	case commandInterrupted(ctx):
		return cancelledOutcome("api")
	case errors.Is(err, probe.ErrUnreachable):
		return checkOutcome{"api", "skip", fmt.Sprintf("%s unreachable (offline?) — not failed", redactBase(base))}
	default:
		return checkOutcome{"api", "fail", err.Error()}
	}
}

// checkSignedIn asks the API as the user who is signed in: the only answer
// that tells a working sign-in from an expired or revoked one. The error of a
// rejected login already says what to do about it. The client is bound to ctx,
// so a Ctrl-C (or the check's own deadline) ends the request, renewal included.
func checkSignedIn(ctx context.Context, creds clientconfig.Creds) checkOutcome {
	base := redactBase(creds.Base)
	c := creds.Client()
	c.SetContext(ctx)
	_, err := c.Whoami()
	var netErr *url.Error
	switch {
	case err == nil:
		return checkOutcome{"api", "ok", fmt.Sprintf("%s reachable, signed in", base)}
	case commandInterrupted(ctx):
		return cancelledOutcome("api")
	case errors.As(err, &netErr):
		return checkOutcome{"api", "skip", fmt.Sprintf("%s unreachable (offline?) — not failed", base)}
	default:
		return checkOutcome{"api", "fail", err.Error()}
	}
}

func checkDB(ctx context.Context) []checkOutcome {
	dsn := repoEnvValue("DATABASE_URL")
	if dsn == "" {
		return []checkOutcome{{"db", "skip", "no DATABASE_URL (no .env in a checkout and none exported)"}}
	}
	if err := probe.DBProbe(ctx, dsn); err != nil {
		if commandInterrupted(ctx) {
			return []checkOutcome{cancelledOutcome("db")}
		}
		// A configured database that cannot be reached is a failure,
		// not an offline skip — it is usually loopback, not internet.
		return []checkOutcome{{"db", "fail", err.Error()}}
	}
	return []checkOutcome{{"db", "ok", "database accepts connections"}}
}

func checkGitHub(ctx context.Context) []checkOutcome {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = repoEnvValue("GITHUB_TOKEN")
	}
	if token == "" {
		return []checkOutcome{{"github", "skip", "no GITHUB_TOKEN — evidence sync runs unauthenticated"}}
	}
	login, err := probe.GitHubProbe(ctx, nil, token)
	switch {
	case err == nil:
		return []checkOutcome{{"github", "ok", fmt.Sprintf("token valid (login %s)", login)}}
	case errors.Is(err, probe.ErrUnreachable):
		return []checkOutcome{{"github", "skip", "api.github.com unreachable (offline?) — not failed"}}
	case errors.Is(err, probe.ErrBadToken), errors.Is(err, probe.ErrRateLimited):
		return []checkOutcome{{"github", "fail", err.Error()}}
	default:
		return []checkOutcome{{"github", "fail", err.Error()}}
	}
}

func checkLLM(ctx context.Context, live bool) []checkOutcome {
	key := repoEnvValue("INSIDEOUT_LLM_API_KEY")
	if key == "" {
		return []checkOutcome{{"llm", "skip", "no INSIDEOUT_LLM_API_KEY — offline coach mode (legal default)"}}
	}
	if err := probe.LLMShape(key); err != nil {
		return []checkOutcome{{"llm", "fail", err.Error()}}
	}
	if !live {
		return []checkOutcome{{"llm", "ok", "key shape ok (live probe: add --llm)"}}
	}
	base := repoEnvValue("INSIDEOUT_LLM_BASE_URL")
	schema := strings.TrimSpace(repoEnvValue("INSIDEOUT_LLM_SCHEMA"))
	if schema == "" {
		schema = "messages" // config.Load's default
	}
	if schema != "openai" {
		return []checkOutcome{{"llm", "skip", fmt.Sprintf("live probe not implemented for schema %q; shape ok", schema)}}
	}
	if base == "" {
		return []checkOutcome{{"llm", "skip", "no INSIDEOUT_LLM_BASE_URL; shape ok"}}
	}
	code, err := llmModelsProbe(ctx, base, key)
	switch {
	case err != nil && commandInterrupted(ctx):
		return []checkOutcome{cancelledOutcome("llm")}
	case err != nil && errors.Is(err, probe.ErrUnreachable):
		return []checkOutcome{{"llm", "skip", "provider unreachable (offline?) — shape ok"}}
	case err != nil:
		return []checkOutcome{{"llm", "fail", err.Error()}}
	default:
		return []checkOutcome{{"llm", "ok", fmt.Sprintf("provider answered /models (HTTP %d)", code)}}
	}
}

// llmModelsProbe makes one cheap authenticated GET {base}/models —
// the OpenAI-compatible way to prove a key without generating tokens.
func llmModelsProbe(ctx context.Context, baseURL, key string) (int, error) {
	url := strings.TrimRight(baseURL, "/") + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", probe.ErrUnreachable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return resp.StatusCode, fmt.Errorf("provider rejected the key (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode >= 500 {
		return resp.StatusCode, fmt.Errorf("provider error (HTTP %d)", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

// repoEnvValue reads one value from the repo .env when a checkout is
// discoverable (INSIDEOUT_ENV_FILE honored); the process environment
// wins, matching how the server itself consumes .env through dev.sh.
func repoEnvValue(name string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	repoRoot, err := findRepoRoot()
	if err != nil {
		return ""
	}
	return readEnvValues(envPathFor(repoRoot))[name]
}

func envPathFor(repoRoot string) string {
	if v := os.Getenv("INSIDEOUT_ENV_FILE"); v != "" {
		return v
	}
	return repoRoot + "/.env"
}

// redactBase shows the scheme+host without any path credentials.
func redactBase(base string) string {
	if i := strings.Index(base, "://"); i >= 0 {
		if j := strings.IndexAny(base[i+3:], "@"); j >= 0 {
			return base[:i+3] + "***" + base[i+3+j:]
		}
	}
	return base
}
