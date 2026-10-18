package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// The front-door tests need the real main() — argv routing, config loading,
// os.Exit codes — exactly as an operator meets it. TestMain re-executes this
// test binary as the CLI when runMainEnv is set, so runCLI is
// `env -i insideout <args>` with a scripted stdin and no hidden inheritance.
const runMainEnv = "INSIDEOUT_TEST_RUN_MAIN"

func TestMain(m *testing.M) {
	if os.Getenv(runMainEnv) == "1" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type cliResult struct {
	stdout, stderr string
	code           int
}

func (r cliResult) output() string { return r.stdout + r.stderr }

// runCLI runs main() in a child process whose whole environment is env
// (plus a throwaway HOME so a developer's own CLI config cannot leak in)
// and returns what the operator would see.
func runCLI(t *testing.T, env []string, stdinScript string, args ...string) cliResult {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append([]string{runMainEnv + "=1", "HOME=" + t.TempDir()}, env...)
	cmd.Stdin = strings.NewReader(stdinScript)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	res := cliResult{stdout: out.String(), stderr: errOut.String()}
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		res.code = exit.ExitCode()
	case err != nil:
		t.Fatalf("run %v: %v", args, err)
	}
	return res
}
