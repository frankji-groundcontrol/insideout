package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
	"golang.org/x/term"
)

// rv4-USER-01: signup is invitation-only by default and only an admin can
// mint an invitation, so on a fresh install `promote-admin <email>` has to
// create the account when it is absent — an UPDATE that matches no row means
// the first admin can never exist. These run the real main() in a child
// process (main_exec_test.go) against real PostgreSQL; the password reaches
// the CLI on stdin only, never argv. DB-gated: a skip counts as red.

func dbEnv(t *testing.T) (url, owner string) {
	t.Helper()
	url, owner = os.Getenv("DATABASE_URL"), os.Getenv("DATABASE_OWNER_URL")
	if url == "" || owner == "" {
		t.Skip("DATABASE_URL and DATABASE_OWNER_URL not both set, skipping integration test")
	}
	return url, owner
}

// serverEnv is the complete environment of the child CLI.
func serverEnv(url, owner string) []string {
	return []string{
		"DATABASE_URL=" + url,
		"DATABASE_OWNER_URL=" + owner,
		"INSIDEOUT_JWT_SECRET=child-process-test-secret-0123456789",
	}
}

func openTestStore(t *testing.T, url string) *store.Store {
	t.Helper()
	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	return st
}

func TestPromoteAdminCreatesTheFirstAdmin(t *testing.T) {
	url, owner := dbEnv(t)
	st := openTestStore(t, url)
	ctx := context.Background()
	email := "first-admin-" + uuid.NewString() + "@example.com"
	const password = "a long passphrase, never on argv"
	if _, err := st.GetUserByEmail(ctx, email); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("precondition: a fresh unique address must not exist yet, got err=%v", err)
	}

	res := runCLI(t, serverEnv(url, owner), password+"\n", "promote-admin", email)

	if res.code != 0 {
		t.Fatalf("promote-admin on an address with no account: exit %d, want 0 (without it a fresh invitation-only install has no first admin)\nstdout: %s\nstderr: %s", res.code, res.stdout, res.stderr)
	}
	u, err := st.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("promote-admin did not create the account: %v", err)
	}
	if u.SystemRole != "admin" {
		t.Fatalf("system_role = %q, want admin", u.SystemRole)
	}
	if ok, err := auth.VerifyPassword(u.PasswordHash, password); err != nil || !ok {
		t.Fatalf("the password read from stdin does not unlock the new admin (ok=%v err=%v)", ok, err)
	}
	for _, secret := range []string{email, password} {
		if strings.Contains(res.output(), secret) {
			t.Fatalf("promote-admin echoed a secret or the address into its output:\n%s", res.output())
		}
	}
}

// A second argument looks like a password to a hurried operator; taking it
// would put the secret in shell history and the process table.
func TestPromoteAdminRefusesAPasswordOnArgv(t *testing.T) {
	url, owner := dbEnv(t)
	st := openTestStore(t, url)
	email := "argv-" + uuid.NewString() + "@example.com"

	res := runCLI(t, serverEnv(url, owner), "a long passphrase\n", "promote-admin", email, "hunter2hunter2")

	if res.code != 2 {
		t.Fatalf("promote-admin <email> <extra>: exit %d, want 2 (usage)\n%s", res.code, res.output())
	}
	if !strings.Contains(res.output(), "stdin") {
		t.Fatalf("the usage error must say the password comes from stdin:\n%s", res.output())
	}
	if _, err := st.GetUserByEmail(context.Background(), email); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("an account was created despite the usage error (err=%v)", err)
	}
}

// A refused password must leave nothing behind — no half-made admin.
func TestPromoteAdminShortPasswordCreatesNothing(t *testing.T) {
	url, owner := dbEnv(t)
	st := openTestStore(t, url)
	email := "short-" + uuid.NewString() + "@example.com"

	res := runCLI(t, serverEnv(url, owner), "short\n", "promote-admin", email)

	if res.code == 0 {
		t.Fatalf("a 5-character password was accepted:\n%s", res.output())
	}
	if !strings.Contains(res.output(), "at least 8") {
		t.Fatalf("the refusal must name the 8-character floor:\n%s", res.output())
	}
	if _, err := st.GetUserByEmail(context.Background(), email); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("an account exists after the password was refused (err=%v)", err)
	}
}

// Promoting an account that exists must not ask for, or replace, its password.
func TestPromoteAdminLeavesAnExistingPasswordAlone(t *testing.T) {
	url, owner := dbEnv(t)
	st := openTestStore(t, url)
	ctx := context.Background()
	email := "existing-" + uuid.NewString() + "@example.com"
	hash, err := auth.HashPassword("the original passphrase")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateUser(ctx, email, hash, "existing-tester"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	res := runCLI(t, serverEnv(url, owner), "", "promote-admin", email)

	if res.code != 0 {
		t.Fatalf("promote-admin on an existing account: exit %d\n%s", res.code, res.output())
	}
	u, err := st.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatal(err)
	}
	if u.SystemRole != "admin" {
		t.Fatalf("system_role = %q, want admin", u.SystemRole)
	}
	if ok, err := auth.VerifyPassword(u.PasswordHash, "the original passphrase"); err != nil || !ok {
		t.Fatalf("promotion replaced the existing password (ok=%v err=%v)", ok, err)
	}
}

// readNewPassword on a pipe: one line, held to the register handler's floor.
func TestReadNewPasswordPiped(t *testing.T) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		t.Skip("stdin is a terminal; the non-TTY path is not exercised here")
	}
	for script, wantErr := range map[string]bool{
		"a long passphrase\n": false,
		"short\n":             true,
		"":                    true,
	} {
		restore := swapStdin(script)
		got, err := readNewPassword()
		restore()
		if (err != nil) != wantErr {
			t.Errorf("readNewPassword(%q) err=%v, want error=%v", script, err, wantErr)
		}
		if err == nil && got != "a long passphrase" {
			t.Errorf("readNewPassword(%q) = %q", script, got)
		}
	}
}

func askNever(t *testing.T) func() (string, error) {
	return func() (string, error) {
		t.Helper()
		t.Error("the password was asked for although the account exists")
		return "", errors.New("unexpected prompt")
	}
}

// Without the owner pool the new account exists but cannot be promoted: the
// error says so, and the re-run (owner pool attached) promotes it without
// asking for a password again — no half-made state to clean up by hand.
func TestPromoteAdminWithoutOwnerPoolSaysSoAndRecovers(t *testing.T) {
	url, owner := dbEnv(t)
	st := openTestStore(t, url)
	ctx := context.Background()
	email := "no-owner-" + uuid.NewString() + "@example.com"
	ask := func() (string, error) { return "a long passphrase", nil }

	created, err := promoteAdmin(ctx, st, email, ask)
	if err == nil || !strings.Contains(err.Error(), "DATABASE_OWNER_URL") {
		t.Fatalf("want an error naming DATABASE_OWNER_URL, got created=%v err=%v", created, err)
	}
	if _, gerr := st.GetUserByEmail(ctx, email); gerr != nil {
		t.Fatalf("the account should exist after the blocked promotion: %v", gerr)
	}

	if aerr := st.AttachMigratePool(ctx, owner); aerr != nil {
		t.Fatal(aerr)
	}
	created, err = promoteAdmin(ctx, st, email, askNever(t))
	if err != nil || created {
		t.Fatalf("re-run: created=%v err=%v, want created=false and no error", created, err)
	}
	if u, _ := st.GetUserByEmail(ctx, email); u == nil || u.SystemRole != "admin" {
		t.Fatalf("re-run did not promote: %+v", u)
	}
}

// Accounts are stored lowercase; a mixed-case, padded argument must land on
// that row instead of minting a second, unreachable one.
func TestPromoteAdminNormalisesTheAddress(t *testing.T) {
	url, owner := dbEnv(t)
	st := openTestStore(t, url)
	ctx := context.Background()
	if err := st.AttachMigratePool(ctx, owner); err != nil {
		t.Fatal(err)
	}
	lower := "mixed-" + uuid.NewString() + "@example.com"
	typed := "  " + strings.ToUpper(lower[:1]) + lower[1:len(lower)-3] + "COM \n"

	created, err := promoteAdmin(ctx, st, typed, func() (string, error) { return "a long passphrase", nil })
	if err != nil || !created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	if u, gerr := st.GetUserByEmail(ctx, lower); gerr != nil || u.SystemRole != "admin" {
		t.Fatalf("the lowercase account is not the admin: %+v %v", u, gerr)
	}
	if _, err := promoteAdmin(ctx, st, "not-an-address", askNever(t)); err == nil {
		t.Fatal("an address without @ must be refused before any prompt")
	}
}
