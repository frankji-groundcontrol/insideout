package main

// The interrupt runner every client verb runs under: Ctrl-C or SIGTERM
// cancels the command's context, and the process ends with status 130.

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// cliCtx is the context of the command now running. Ctrl-C or SIGTERM cancels
// it (runInterruptible) and every client a verb builds is bound to it, so the
// request in flight ends with the command. It is context.Background when no
// command is running through runInterruptible, as in a test that calls a verb.
var cliCtx = context.Background()

// interruptGrace is how long an interrupted command gets to wind down (its
// request aborts, its error prints) before the process ends anyway. A command
// that waits at a prompt has no request to abort.
var interruptGrace = 400 * time.Millisecond

// runInterruptible is runClientCommand under cliCtx. The first Ctrl-C (or
// SIGTERM) cancels the context and the command ends with status 130, the
// shell's code for an interrupted command; a command still running after
// interruptGrace, or a second signal, ends the process at once. The one thing
// an interrupt must not cut is the renewal of an expired session: the server
// rotates the refresh token as it reads the request, so a process that ends
// before the new pair is saved leaves the saved login holding a spent token.
// apiclient lets a renewal in flight finish (renewalDrain) and the forced exit
// waits for it (WaitForRenewal); only an answer slower than that, or a second
// signal, can still lose the pair (review-4 QUAL-04, rv4-FR-F66).
func runInterruptible(args []string) (handled bool, exitCode int) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	cliCtx = ctx
	finished := make(chan struct{})
	go func() {
		select {
		case <-sigs:
			cancel()
			signal.Stop(sigs) // the next signal takes the default action
			select {
			case <-finished:
			case <-time.After(interruptGrace):
				apiclient.WaitForRenewal()
				fmt.Fprintln(os.Stderr, "\ninsideout: interrupted")
				os.Exit(130)
			}
		case <-finished:
		}
	}()
	handled, exitCode = runClientCommand(args)
	interrupted := ctx.Err() != nil
	close(finished)
	signal.Stop(sigs)
	cancel()
	cliCtx = context.Background()
	if handled && interrupted && exitCode != 0 {
		fmt.Fprintln(os.Stderr, "insideout: interrupted")
		return true, 130
	}
	return handled, exitCode
}
