package main

// Local-mode assembly for `insideout setup --local` (slice 2, the
// B+D+E join): repo discovery, .env writing through the Go catalog
// writer, bundled-Docker or own-URL Postgres, a native pgx probe,
// in-process migrate (with the owner-pool branch), a boot offer that
// injects the written values into the child's environment, and a
// summary gate. Values read from or written to .env are never
// printed; everything a user sees goes through catalog.Mask or
// catalog.RedactDSN or is not a value at all.

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/setup/catalog"
	"github.com/frankji-groundcontrol/insideout/server/internal/setup/envwrite"
	"github.com/frankji-groundcontrol/insideout/server/internal/setup/probe"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// runLocalSetup executes the local branch. postgres is "bundled" or
// "own"; noStart mirrors scripts/setup.sh's --no-start (write .env
// only). It speaks only through the returned exit code and
// stderr/stdout lines that carry no secret values.
func runLocalSetup(postgres string, noStart bool) (bool, int) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup: %v (local mode needs the checkout: docker-compose.yml, .env.example, migrations)\n", err)
		return true, 2
	}
	envPath := filepath.Join(repoRoot, ".env")
	if v := os.Getenv("INSIDEOUT_ENV_FILE"); v != "" {
		envPath = v // test affordance, same override env.sh honors
	}
	examplePath := filepath.Join(repoRoot, ".env.example")

	if err := envwrite.EnsureFromExample(envPath, examplePath); err != nil {
		fmt.Fprintf(os.Stderr, "setup: prepare .env: %v\n", err)
		return true, 1
	}
	vals := readEnvValues(envPath)

	// JWT secret: keep a usable one, rotate placeholders and short
	// values (mirrors scripts/setup.sh's rule).
	jwt := vals["INSIDEOUT_JWT_SECRET"]
	if jwtNeedsRotation(jwt) {
		secret, err := generateJWTSecret()
		if err != nil {
			fmt.Fprintf(os.Stderr, "setup: generate JWT secret: %v\n", err)
			return true, 1
		}
		if err := envwrite.SetVar(envPath, "INSIDEOUT_JWT_SECRET", secret); err != nil {
			fmt.Fprintf(os.Stderr, "setup: write INSIDEOUT_JWT_SECRET: %v\n", err)
			return true, 1
		}
		fmt.Fprintln(os.Stderr, "setup: generated INSIDEOUT_JWT_SECRET (not printed)")
	} else {
		fmt.Fprintln(os.Stderr, "setup: INSIDEOUT_JWT_SECRET already set")
	}

	var dsn string
	switch postgres {
	case "own":
		if len(os.Args) > 0 && !isTerminal(os.Stdin) {
			// Agent/piped path: the URL arrives on stdin (setup.sh parity).
			line, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil && line == "" {
				fmt.Fprintln(os.Stderr, "setup: empty DATABASE_URL on stdin")
				return true, 2
			}
			dsn = strings.TrimSpace(line)
		} else {
			secret, err := readSecret("DATABASE_URL (input hidden): ")
			if err != nil {
				fmt.Fprintln(os.Stderr, "setup:", err)
				return true, 2
			}
			dsn = secret
		}
		if dsn == "" {
			fmt.Fprintln(os.Stderr, "setup: empty DATABASE_URL")
			return true, 2
		}
	case "bundled":
		dsn = bundledDSN(vals["POSTGRES_PORT"], vals["POSTGRES_APP_PASSWORD"])
	default:
		fmt.Fprintln(os.Stderr, "setup: --local needs --postgres bundled|own")
		return true, 2
	}
	if err := envwrite.SetVar(envPath, "DATABASE_URL", dsn); err != nil {
		fmt.Fprintf(os.Stderr, "setup: write DATABASE_URL: %v\n", err)
		return true, 1
	}
	fmt.Fprintf(os.Stderr, "setup: wrote DATABASE_URL → %s\n", catalog.RedactDSN(dsn))

	if noStart {
		fmt.Fprintln(os.Stderr, "setup: --no-start: .env updated; database not started, migrations not applied")
		if err := printSummaryGate(envPath, examplePath); err != nil {
			fmt.Fprintf(os.Stderr, "setup: summary: %v\n", err)
		}
		return true, 0
	}

	ctx := context.Background()
	if postgres == "bundled" {
		if err := composeUpPostgres(repoRoot, envPath); err != nil {
			fmt.Fprintf(os.Stderr, "setup: bundled Postgres: %v\n", err)
			return true, 1
		}
	}
	if err := waitForDatabase(ctx, dsn, 30*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "setup: database not reachable: %v\n", err)
		return true, 1
	}
	fmt.Fprintln(os.Stderr, "setup: database accepts connections")

	st, err := store.Open(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup: open store: %v\n", err)
		return true, 1
	}
	if ownerURL := vals["DATABASE_OWNER_URL"]; ownerURL != "" {
		if err := st.AttachMigratePool(ctx, ownerURL); err != nil {
			st.Close()
			fmt.Fprintf(os.Stderr, "setup: owner migrate pool: %v\n", err)
			return true, 1
		}
	}
	applied, err := st.Migrate(ctx)
	st.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup: migrate: %v\n", err)
		return true, 1
	}
	fmt.Fprintf(os.Stderr, "setup: migrations applied (%d)\n", len(applied))

	if err := printSummaryGate(envPath, examplePath); err != nil {
		fmt.Fprintf(os.Stderr, "setup: summary: %v\n", err)
	}
	fmt.Fprintln(os.Stderr, "\nThe installed CLI keeps calling the hosted API until you opt into this server:")
	fmt.Fprintln(os.Stderr, "  export INSIDEOUT_API=http://127.0.0.1:8080/api/v1")

	return offerBoot(repoRoot, envPath)
}

// offerBoot starts the API in the foreground when the user confirms
// at a terminal; otherwise it prints the start command (dev.sh loads
// .env without printing it). The child process must receive the
// written values in its environment — the server reads only process
// env (config.Load), not .env.
func offerBoot(repoRoot, envPath string) (bool, int) {
	vals := readEnvValues(envPath)
	if isTerminal(os.Stdin) {
		fmt.Print("Start the API now? [y/N] ")
		var answer string
		fmt.Scanln(&answer)
		if strings.HasPrefix(strings.ToLower(answer), "y") {
			child := exec.Command(selfPath())
			child.Dir = repoRoot
			child.Env = childEnv(os.Environ(), vals)
			child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
			if err := child.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "setup: server exited: %v\n", err)
				return true, 1
			}
			return true, 0
		}
	}
	fmt.Fprintln(os.Stderr, "Start the API:")
	fmt.Fprintln(os.Stderr, "  ./scripts/dev.sh -C server go run ./cmd/insideout")
	fmt.Fprintln(os.Stderr, "First admin: ./scripts/dev.sh -C server go run ./cmd/insideout promote-admin you@example.com")
	return true, 0
}

func selfPath() string {
	if len(os.Args) > 0 && os.Args[0] != "" {
		return os.Args[0]
	}
	return "insideout"
}

// findRepoRoot walks up from the working directory to a checkout
// (docker-compose.yml + .env.example + server/go.mod).
// INSIDEOUT_SETUP_REPO_ROOT overrides — the test affordance — but the
// override must still be a real checkout.
func findRepoRoot() (string, error) {
	if v := os.Getenv("INSIDEOUT_SETUP_REPO_ROOT"); v != "" {
		if !repoMarker(v) {
			return "", fmt.Errorf("INSIDEOUT_SETUP_REPO_ROOT=%s is not a repository checkout", v)
		}
		return v, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if repoMarker(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no repository checkout found above %s", dir)
		}
		dir = parent
	}
}

func repoMarker(dir string) bool {
	for _, name := range []string{"docker-compose.yml", ".env.example", filepath.Join("server", "go.mod")} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return false
		}
	}
	return true
}

// readEnvValues reads KEY=VALUE lines (last wins, comments skipped).
// Values feed machine-to-machine assembly (DSN construction, child
// env) and are never printed.
func readEnvValues(path string) map[string]string {
	vals := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return vals
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ") // dotenv-tolerant
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if k != "" {
			vals[k] = v
		}
	}
	return vals
}

// bundledDSN mirrors scripts/setup.sh: app role, loopback, sslmode
// off, port/password from .env with the documented dev defaults.
func bundledDSN(port, password string) string {
	if port == "" {
		port = "5442"
	}
	if password == "" {
		password = "insideout_app_dev_password"
	}
	return fmt.Sprintf("postgres://insideout_app:%s@127.0.0.1:%s/insideout?sslmode=disable", password, port)
}

// jwtNeedsRotation mirrors scripts/setup.sh: placeholder, or shorter
// than 32 characters, is not a usable secret.
func jwtNeedsRotation(s string) bool {
	return s == "" || s == "change_me_to_a_long_random_string" || len(s) < 32
}

func generateJWTSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// childEnv returns an environ for the server child: the current
// environment with the setup-written values overriding, plus
// INSIDEOUT_COOKIE_SECURE=0 because a local server serves plain http
// (the value .env.example documents for exactly this case).
func childEnv(environ []string, vals map[string]string) []string {
	override := map[string]string{
		"DATABASE_URL":            vals["DATABASE_URL"],
		"INSIDEOUT_JWT_SECRET":    vals["INSIDEOUT_JWT_SECRET"],
		"INSIDEOUT_COOKIE_SECURE": "0",
	}
	var out []string
	for _, kv := range environ {
		key, _, ok := strings.Cut(kv, "=")
		if ok && override[key] != "" {
			continue
		}
		out = append(out, kv)
	}
	for k, v := range override {
		if v != "" {
			out = append(out, k+"="+v)
		}
	}
	return out
}

// composeUpPostgres starts the bundled database. --env-file points
// compose at the same .env the writer produced so a scratch
// INSIDEOUT_ENV_FILE drives a scratch port.
func composeUpPostgres(repoRoot, envPath string) error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker is required for bundled Postgres (brew install --cask docker, or use --postgres own)")
	}
	cmd := exec.Command("docker", "compose", "--env-file", envPath, "up", "-d", "postgres")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, lastLines(string(out), 5))
	}
	return nil
}

// waitForDatabase pings until the deadline — a freshly created
// bundled container needs seconds before it accepts connections.
func waitForDatabase(ctx context.Context, dsn string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		err := probe.DBProbe(ctx, dsn)
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return err
		}
		time.Sleep(time.Second)
	}
}

// printSummaryGate is the final gate: what is set, what rides a
// default, what is still outstanding — names only, never values.
func printSummaryGate(envPath, examplePath string) error {
	example, err := os.ReadFile(examplePath)
	if err != nil {
		return err
	}
	vars, err := catalog.ParseSkeleton(example)
	if err != nil {
		return err
	}
	env, err := os.ReadFile(envPath)
	if err != nil {
		return err
	}
	states := catalog.Evaluate(env, vars)
	counts := map[catalog.Status]int{}
	var outstanding []string
	for _, s := range states {
		counts[s.Status]++
		if s.Var.Required && (s.Status == catalog.StatusMissing || s.Status == catalog.StatusPlaceholder) {
			outstanding = append(outstanding, s.Var.Name)
		}
	}
	fmt.Fprintf(os.Stderr, "\nSummary: %d set · %d default · %d placeholder · %d missing\n",
		counts[catalog.StatusSet], counts[catalog.StatusDefault],
		counts[catalog.StatusPlaceholder], counts[catalog.StatusMissing]+counts[catalog.StatusUnset])
	if len(outstanding) > 0 {
		fmt.Fprintf(os.Stderr, "Required still outstanding: %s\n", strings.Join(outstanding, ", "))
	} else {
		fmt.Fprintln(os.Stderr, "Required: all satisfied")
	}
	return nil
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
