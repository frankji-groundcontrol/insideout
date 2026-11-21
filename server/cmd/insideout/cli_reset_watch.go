package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// confirmMailSend asks before a reset email is sent; anything but y is no.
func confirmMailSend(email string) bool {
	fmt.Fprintf(os.Stderr, "Send email to %s? [y/N] ", email)
	var answer string
	fmt.Scanln(&answer)
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(answer)), "y")
}

// watchReset polls until the reset link is used, the watch expires, or
// the command is interrupted. completed means the password was changed
// somewhere else, so the saved login for this API is dropped.
func watchReset(ctx context.Context, c *apiclient.Client, base, id string) (bool, int) {
	fmt.Fprintln(os.Stderr, "waiting until the link is used — finish in the browser, or Ctrl-C and run `insideout reset-password`")
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		status, err := c.ResetWatchStatus(id)
		if err != nil {
			if ctx.Err() != nil {
				return true, 1
			}
			fmt.Fprintln(os.Stderr, err)
			return true, 1
		}
		switch status {
		case "completed":
			if err := forgetSavedLogin(base); err != nil {
				fmt.Fprintf(os.Stderr, "insideout: the password was reset, but the saved login could not be removed (%v)\n", err)
			}
			fmt.Fprintln(os.Stderr, "password reset somewhere else; every session was revoked — sign in again")
			return true, 0
		case "expired":
			fmt.Fprintln(os.Stderr, "the reset link expired before it was used")
			return true, 1
		}
		select {
		case <-ctx.Done():
			return true, 1
		case <-tick.C:
		}
	}
}
