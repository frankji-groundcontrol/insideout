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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/setup/catalog"
	"github.com/frankji-groundcontrol/insideout/server/internal/setup/envwrite"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// runLocalSetup executes the local branch. postgres is "bundled" or
// "own"; noStart mirrors scripts/setup.sh's --no-start (write .env
// only); createRoles lets the own-URL path create missing database
// roles without asking (the agent path, where stdin is the DSN). It
// speaks only through the returned exit code and stderr/stdout lines
// that carry no secret values.
func runLocalSetup(postgres string, noStart, createRoles bool) (bool, int) {
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
	ownerURL := vals["DATABASE_OWNER_URL"]
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
		// The bundled app role cannot apply migrations, so the owner DSN the
		// init script's other role makes possible travels with it.
		ownerURL = bundledOwnerDSN(vals["POSTGRES_PORT"], vals["POSTGRES_OWNER_PASSWORD"])
	default:
		fmt.Fprintln(os.Stderr, "setup: --local needs --postgres bundled|own")
		return true, 2
	}
	// An own-URL DSN becomes the runtime URL once the role gate below has
	// passed (ensureRoles writes it): a stop before that must not leave an
	// admin DSN in .env for `serve` to run as. --no-start never reaches the
	// gate, so it writes now.
	if postgres == "bundled" || noStart {
		if !wireDatabaseURL(envPath, dsn) {
			return true, 1
		}
	}
	if postgres == "bundled" {
		if err := envwrite.SetVar(envPath, "DATABASE_OWNER_URL", ownerURL); err != nil {
			fmt.Fprintf(os.Stderr, "setup: write DATABASE_OWNER_URL: %v\n", err)
			return true, 1
		}
		fmt.Fprintf(os.Stderr, "setup: wrote DATABASE_OWNER_URL → %s\n", catalog.RedactDSN(ownerURL))
	}

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
	if postgres == "own" {
		// A fresh own-URL database lacks the two roles the migrations
		// need; find out now, not mid-migration.
		var ok bool
		if ownerURL, ok = ensureRoles(ctx, st.Pool, dsn, envPath, ownerURL, createRoles); !ok {
			st.Close()
			return true, 1
		}
	}
	if ownerURL != "" {
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

// wireDatabaseURL writes the runtime URL into .env and says so, redacted.
func wireDatabaseURL(envPath, dsn string) bool {
	if err := envwrite.SetVar(envPath, "DATABASE_URL", dsn); err != nil {
		fmt.Fprintf(os.Stderr, "setup: write DATABASE_URL: %v\n", err)
		return false
	}
	fmt.Fprintf(os.Stderr, "setup: wrote DATABASE_URL → %s\n", catalog.RedactDSN(dsn))
	return true
}

// bundledDSN mirrors scripts/setup.sh: app role, loopback, sslmode
// off, port/password from .env with the documented dev defaults.
func bundledDSN(port, password string) string {
	return bundledRoleDSN(appRole, port, password, "insideout_app_dev_password")
}

// bundledOwnerDSN is the same bundled database as insideout_owner, the
// role migrations run as (POSTGRES_OWNER_PASSWORD in docker-compose.yml).
func bundledOwnerDSN(port, password string) string {
	return bundledRoleDSN(ownerRole, port, password, "insideout_owner_dev_password")
}

func bundledRoleDSN(role, port, password, devPassword string) string {
	if port == "" {
		port = "5442"
	}
	if password == "" {
		password = devPassword
	}
	return fmt.Sprintf("postgres://%s:%s@127.0.0.1:%s/insideout?sslmode=disable", role, password, port)
}
