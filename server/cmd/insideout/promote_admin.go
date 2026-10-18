package main

// promote-admin — the only path to admin (there is deliberately no
// self-promotion API). Signup is invitation-only by default and only an
// admin can mint an invitation, so on a fresh install the first admin has
// no account to promote: an absent account is created here, with a
// password read from stdin (never argv), and then promoted. rv4-USER-01.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"golang.org/x/term"
)

// Same address rule and password floor as the register handler
// (internal/api/auth.go); the pattern is unexported there.
var adminEmailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

const minAdminPasswordLen = 8

// runPromoteAdmin sets system_role='admin' for one account, creating it
// first when it does not exist. Cross-user updates on users are RLS-blocked
// for the app role, so the role change needs the owner pool
// (DATABASE_OWNER_URL).
func runPromoteAdmin(ctx context.Context, log *slog.Logger, st *store.Store, email string) {
	created, err := promoteAdmin(ctx, st, email, readNewPassword)
	if err != nil {
		log.Error("promote-admin", "error", err)
		os.Exit(1)
	}
	// The operator typed the address; the log may end up in CI or hosted
	// log storage, so it states the fact without echoing it.
	log.Info("promote-admin: done", "role", "admin", "created", created)
}

// promoteAdmin reports whether it had to create the account. newPassword is
// asked only then: promoting an account that exists never touches its
// password.
func promoteAdmin(ctx context.Context, st *store.Store, email string, newPassword func() (string, error)) (created bool, err error) {
	// Register and login both store and match lowercase, so a mixed-case
	// argument must find (or create) the lowercase row.
	email = strings.ToLower(strings.TrimSpace(email))
	_, err = st.GetUserByEmail(ctx, email)
	switch {
	case errors.Is(err, store.ErrNotFound):
		if created, err = createAdminAccount(ctx, st, email, newPassword); err != nil {
			return false, err
		}
	case err != nil:
		return false, err
	}
	if err := st.SetSystemRole(ctx, email, "admin"); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// The account is there, yet the update matched no row: without
			// the owner pool RLS hides other users from the app role. Re-running
			// after the fix promotes the account without asking for a password.
			return created, errors.New("the account exists but the role change was blocked; set DATABASE_OWNER_URL (the owner pool) and run promote-admin again")
		}
		return created, err
	}
	return created, nil
}

func createAdminAccount(ctx context.Context, st *store.Store, email string, newPassword func() (string, error)) (bool, error) {
	if !adminEmailPattern.MatchString(email) {
		return false, errors.New("not a valid email address")
	}
	password, err := newPassword()
	if err != nil {
		return false, err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return false, err
	}
	if _, err := st.CreateUser(ctx, email, hash, emailLocalPart(email)); err != nil {
		return false, err
	}
	return true, nil
}

// readNewPassword asks for the new admin's password: hidden and typed twice
// on a terminal (a typo must not lock out the only admin), one line from a
// pipe so agents, scripts and tests keep working.
func readNewPassword() (string, error) {
	password, err := readSecret("new admin password (min 8 characters): ")
	if err != nil {
		return "", err
	}
	if len(password) < minAdminPasswordLen {
		return "", fmt.Errorf("the password must be at least %d characters", minAdminPasswordLen)
	}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		again, err := readSecret("repeat password: ")
		if err != nil {
			return "", err
		}
		if again != password {
			return "", errors.New("the two passwords differ")
		}
	}
	return password, nil
}
