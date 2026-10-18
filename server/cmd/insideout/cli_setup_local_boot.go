package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/setup/catalog"
	"github.com/frankji-groundcontrol/insideout/server/internal/setup/probe"
)

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
	fmt.Fprintln(os.Stderr, "First admin on a shared server: promote an existing account with the owner DSN")
	fmt.Fprintln(os.Stderr, "Disposable local database: run `insideout seed`, then promote demo@insideout.local")
	return true, 0
}

func selfPath() string {
	if len(os.Args) > 0 && os.Args[0] != "" {
		return os.Args[0]
	}
	return "insideout"
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
