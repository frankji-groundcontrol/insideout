package local

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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/setup/catalog"
	"github.com/frankji-groundcontrol/insideout/server/internal/setup/envwrite"
	"github.com/frankji-groundcontrol/insideout/server/internal/setup/ui"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// ExitBack and ExitExit are returned only from a terminal prompt, so
// the interactive wizard can leave a step. The flag-driven path does
// not use them.
const (
	ExitBack = 3
	ExitExit = 4
)

// Options is one local run. Postgres is "bundled" or "own"; NoStart
// mirrors the old scripts/setup.sh --no-start (write .env only);
// CreateRoles lets the own-URL path create missing database roles
// without asking (the agent path, where stdin is the DSN); OwnDSN is
// the URL the wizard already collected; AskBoot offers to start the
// API at the end. Prompt and ReadSecret come from the CLI so every
// prompt of one command shares its stdin reader.
type Options struct {
	Postgres             string
	NoStart, CreateRoles bool
	OwnDSN               string
	AskBoot              bool
	Prompt               func(label string) string
	ReadSecret           func(label string) (string, error)
}

// run carries one Run's state from step to step.
type run struct {
	Options
	repoRoot, envPath, examplePath string
	vals                           map[string]string
	dsn, ownerURL                  string
}

// Run executes the local branch. It speaks only through the returned
// exit code and stderr/stdout lines that carry no secret values.
func Run(o Options) int {
	r := &run{Options: o}
	if code := r.prepareEnv(); code >= 0 {
		return code
	}
	if code := r.resolveDSN(); code >= 0 {
		return code
	}
	if code := r.wireEarly(); code >= 0 {
		return code
	}
	if r.NoStart {
		fmt.Fprintln(os.Stderr, "setup: --no-start: .env updated; database not started, migrations not applied")
		if err := printSummaryGate(r.envPath, r.examplePath); err != nil {
			fmt.Fprintf(os.Stderr, "setup: summary: %v\n", err)
		}
		return 0
	}
	if code := r.migrate(context.Background()); code >= 0 {
		return code
	}
	if err := printSummaryGate(r.envPath, r.examplePath); err != nil {
		fmt.Fprintf(os.Stderr, "setup: summary: %v\n", err)
	}
	fmt.Fprintln(os.Stderr, "\nThe installed CLI keeps calling the hosted API until you opt into this server:")
	fmt.Fprintln(os.Stderr, "  export INSIDEOUT_API=http://127.0.0.1:8080/api/v1")
	if !r.AskBoot {
		return 0
	}
	return offerBoot(r.repoRoot, r.envPath)
}

// prepareEnv finds the checkout, creates .env from the skeleton when
// absent and rotates an unusable JWT secret. Each step returns -1 to
// continue, else the exit code.
func (r *run) prepareEnv() int {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup: %v (local mode needs the checkout: docker-compose.yml, .env.example, migrations)\n", err)
		return 2
	}
	r.repoRoot, r.envPath = repoRoot, EnvPath(repoRoot)
	r.examplePath = filepath.Join(repoRoot, ".env.example")

	if err := envwrite.EnsureFromExample(r.envPath, r.examplePath); err != nil {
		fmt.Fprintf(os.Stderr, "setup: prepare .env: %v\n", err)
		return 1
	}
	r.vals = ReadEnvValues(r.envPath)

	// JWT secret: keep a usable one, rotate placeholders and short
	// values (the rule the old scripts/setup.sh had).
	if !jwtNeedsRotation(r.vals["INSIDEOUT_JWT_SECRET"]) {
		fmt.Fprintln(os.Stderr, "setup: INSIDEOUT_JWT_SECRET already set")
		return -1
	}
	secret, err := generateJWTSecret()
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup: generate JWT secret: %v\n", err)
		return 1
	}
	if err := envwrite.SetVar(r.envPath, "INSIDEOUT_JWT_SECRET", secret); err != nil {
		fmt.Fprintf(os.Stderr, "setup: write INSIDEOUT_JWT_SECRET: %v\n", err)
		return 1
	}
	fmt.Fprintln(os.Stderr, "setup: generated INSIDEOUT_JWT_SECRET (not printed)")
	return -1
}

// resolveDSN sets the runtime DSN (and, bundled, the owner DSN).
func (r *run) resolveDSN() int {
	r.ownerURL = r.vals["DATABASE_OWNER_URL"]
	switch r.Postgres {
	case "own":
		return r.ownDSN()
	case "bundled":
		r.dsn = bundledDSN(r.vals["POSTGRES_PORT"], r.vals["POSTGRES_APP_PASSWORD"])
		// The bundled app role cannot apply migrations, so the owner DSN the
		// init script's other role makes possible travels with it.
		r.ownerURL = bundledOwnerDSN(r.vals["POSTGRES_PORT"], r.vals["POSTGRES_OWNER_PASSWORD"])
		return -1
	default:
		fmt.Fprintln(os.Stderr, "setup: --local needs --postgres bundled|own")
		return 2
	}
}

// ownDSN takes the own URL from the wizard, a pipe or a hidden prompt.
func (r *run) ownDSN() int {
	switch {
	case r.OwnDSN != "":
		r.dsn = r.OwnDSN
	case len(os.Args) > 0 && !ui.IsTerminal(os.Stdin):
		// Agent/piped path: the URL arrives on stdin.
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			fmt.Fprintln(os.Stderr, "setup: empty DATABASE_URL on stdin")
			return 2
		}
		r.dsn = strings.TrimSpace(line)
	default:
		secret, err := r.ReadSecret("DATABASE_URL (input hidden, b = back, q = exit): ")
		if err != nil {
			fmt.Fprintln(os.Stderr, "setup:", err)
			return 2
		}
		switch strings.ToLower(strings.TrimSpace(secret)) {
		case "b", "back":
			return ExitBack
		case "q", "exit":
			return ExitExit
		}
		r.dsn = secret
	}
	if r.dsn == "" {
		fmt.Fprintln(os.Stderr, "setup: empty DATABASE_URL")
		return 2
	}
	return -1
}

// wireEarly writes what may be written before the database is seen.
// An own-URL DSN becomes the runtime URL once the role gate has passed
// (ensureRoles writes it): a stop before that must not leave an admin
// DSN in .env for `serve` to run as. --no-start never reaches the
// gate, so it writes now.
func (r *run) wireEarly() int {
	if r.Postgres == "bundled" || r.NoStart {
		if !wireDatabaseURL(r.envPath, r.dsn) {
			return 1
		}
	}
	if r.Postgres == "bundled" {
		if err := envwrite.SetVar(r.envPath, "DATABASE_OWNER_URL", r.ownerURL); err != nil {
			fmt.Fprintf(os.Stderr, "setup: write DATABASE_OWNER_URL: %v\n", err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "setup: wrote DATABASE_OWNER_URL → %s\n", catalog.RedactDSN(r.ownerURL))
	}
	return -1
}

// migrate starts the bundled database, waits for it, gates the own
// URL's roles and applies the migrations.
func (r *run) migrate(ctx context.Context) int {
	if r.Postgres == "bundled" {
		if err := composeUpPostgres(r.repoRoot, r.envPath); err != nil {
			fmt.Fprintf(os.Stderr, "setup: bundled Postgres: %v\n", err)
			return 1
		}
	}
	if ui.IsTerminal(os.Stdin) {
		fmt.Fprintln(os.Stderr, "\n\033[1m1.3.3\033[2m · \033[0mChecking the database …")
	} else {
		fmt.Fprintln(os.Stderr, "Checking the database …")
	}
	if err := waitForDatabase(ctx, r.dsn, 30*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "Not reachable: %v\nFix the database settings before continuing.\n", err)
		return 1
	}
	fmt.Fprintln(os.Stderr, "Reachable.")

	st, err := store.Open(ctx, r.dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup: open store: %v\n", err)
		return 1
	}
	defer st.Close()
	if r.Postgres == "own" {
		// A fresh own-URL database lacks the two roles the migrations
		// need; find out now, not mid-migration.
		var ok bool
		if r.ownerURL, ok = ensureRoles(ctx, st.Pool, r.dsn, r.envPath, r.ownerURL, r.CreateRoles, r.Prompt); !ok {
			return 1
		}
	}
	if r.ownerURL != "" {
		if err := st.AttachMigratePool(ctx, r.ownerURL); err != nil {
			fmt.Fprintf(os.Stderr, "setup: owner migrate pool: %v\n", err)
			return 1
		}
	}
	applied, err := st.Migrate(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup: migrate: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "setup: migrations applied (%d)\n", len(applied))
	return -1
}
