package main

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// rv4-DATA-04: every policy in db/migrations is bypassed by a pool role with
// BYPASSRLS or SUPERUSER, and FORCE ROW LEVEL SECURITY does not stop either.
// The server has to refuse to serve on such a role rather than run with row
// security quietly off. The real main() runs in a child process, as an
// operator meets it (main_exec_test.go), against real PostgreSQL: DATABASE_URL
// is the insideout_owner DSN, whose role has BYPASSRLS. DB-gated: a skip
// counts as red.
//
// The child listens on an ephemeral loopback port, so a server that wrongly
// starts holds no fixed port; the deadline kills it and fails the test.
func TestServeRefusesAPoolRoleThatBypassesRLS(t *testing.T) {
	_, owner := dbEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0]) // no verb: serve
	cmd.Env = append([]string{runMainEnv + "=1", "HOME=" + t.TempDir(), "INSIDEOUT_ADDR=127.0.0.1:0"}, serverEnv(owner, owner)...)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()

	if ctx.Err() != nil {
		t.Fatalf("serve was still running 20s after start on a pool role with BYPASSRLS: it must refuse to start\n%s", out.String())
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 {
		t.Fatalf("serve on a pool role with BYPASSRLS: want exit 1, got %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "BYPASSRLS") || !strings.Contains(out.String(), "DATABASE_URL") {
		t.Errorf("the refusal must name the role attribute and DATABASE_URL, got:\n%s", out.String())
	}

	// A startup error reaches CI and hosted logs: no DSN, password or host.
	u, perr := url.Parse(owner)
	if perr != nil {
		t.Fatalf("parse owner url: %v", perr)
	}
	password, _ := u.User.Password()
	for _, secret := range []string{owner, password, u.Host} {
		if secret != "" && strings.Contains(out.String(), secret) {
			t.Errorf("the refusal printed part of the DSN:\n%s", out.String())
		}
	}
}
