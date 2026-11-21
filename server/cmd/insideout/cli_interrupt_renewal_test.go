package main

import (
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// Review-4 FR-F66. runInterruptible promised that an interrupt cannot land
// between the server rotating the refresh token and the CLI saving the new
// pair. It could: the context it cancels was bound to the renewal POST, and the
// forced exit after interruptGrace ended the process mid-renewal. Either way the
// saved login held a refresh token the server had already spent, and the next
// command told the user to sign in again.

// rvApiClientInterruptDuringRenewal runs `whoami` on an expired saved login whose
// renewal answers answerAfter after it arrives, sends SIGINT when it has
// arrived, and requires the rotated pair to be on disk once the command is gone.
// grace is the child's interruptGrace.
func rvApiClientInterruptDuringRenewal(t *testing.T, answerAfter time.Duration, grace string) {
	t.Helper()
	arrived := make(chan struct{}, 1)
	answered := make(chan struct{})
	var rotated atomic.Bool
	api := newExpiredAPI(t, func(w http.ResponseWriter, a *expiredAPI, body string) {
		defer close(answered)
		arrived <- struct{}{}
		time.Sleep(answerAfter)
		rotated.Store(strings.Contains(body, `"ref-old"`))
		swapPair(w, a, body)
	})
	old := clientconfig.Config{APIBase: api.URL, Token: "acc-old", RefreshToken: "ref-old"}
	writeClientConfig(t, old)

	// The child reads the saved login, not a token from its environment.
	cmd, errOut := startCLIWithEnv(t, cliEnv("XDG_CONFIG_HOME=", "INSIDEOUT_TOKEN=", graceEnv+"="+grace), "", "whoami")
	select {
	case <-arrived:
	case <-time.After(10 * time.Second):
		t.Fatalf("the renewal never reached the API (stderr %q)", errOut.String())
	}
	wantInterrupted(t, cmd, errOut)

	select {
	case <-answered:
	case <-time.After(5 * time.Second):
		t.Fatal("the API never finished answering the renewal")
	}
	if !rotated.Load() {
		t.Fatal("the API did not rotate ref-old: the test set nothing up")
	}
	if got, want := savedConfig(t), (clientconfig.Config{APIBase: api.URL, Token: "acc-new", RefreshToken: "ref-new"}); got != want {
		t.Errorf("saved config after the interrupt = %+v, want the pair the server rotated to %+v", got, want)
	}
}

func TestRvApiClientInterruptedRenewalStillSavesTheRotatedPair(t *testing.T) {
	// The grace is an hour, so only the cancelled context can end the command:
	// the renewal in flight must survive the cancellation and finish its save.
	t.Run("the command ends by itself", func(t *testing.T) {
		rvApiClientInterruptDuringRenewal(t, 300*time.Millisecond, "1h")
	})
	// The answer comes after interruptGrace (100 ms) but well inside the drain
	// (750 ms): the process may not be ended before the pair that arrives is saved.
	t.Run("the grace runs out first", func(t *testing.T) {
		rvApiClientInterruptDuringRenewal(t, 400*time.Millisecond, "100ms")
	})
}
