package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Review-4 QUAL-04, repair round 1. cli_ctx_signal_test.go starts the real
// binary, and a client built without cliCtx never sees the signal, yet the
// command still ended with status 130 once interruptGrace ran out: those tests
// stayed green with the context threading removed. These tests leave the signal
// and the grace out. They make a cancellable context the cliCtx in force, cancel
// it while a request is in flight and require that request to end because of
// it, once for each place the CLI builds an API client:
//
//	clientFromFlag    cli.go        every verb that uses the saved login
//	runAuthCommand    cli_auth.go   login, register, forgot-password, reset-password
//	setupClient       cli_setup.go  the setup form's sign-in
//
// Deleting the SetContext(cliCtx) call at any of them turns its case red.

// holdingAPI tells which requests reach it and holds each one until its client
// leaves; the backstop only keeps a regression from hanging the suite.
func holdingAPI(t *testing.T) (url string, arrived <-chan struct{}) {
	t.Helper()
	in := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body) // the server notices a client that left only once the body is read
		select {
		case in <- struct{}{}:
		default:
		}
		select {
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
			w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL, in
}

// setCommandContext makes ctx the cliCtx in force for the test, as
// runInterruptible does for the command it runs.
func setCommandContext(t *testing.T, ctx context.Context) {
	t.Helper()
	old := cliCtx
	cliCtx = ctx
	t.Cleanup(func() { cliCtx = old })
}

// cancelOnArrival makes a cancellable cliCtx and cancels it the moment a request
// reaches the API, as a Ctrl-C does. Should none ever arrive it cancels after
// 5 s, so a client that never calls cannot hang the test.
func cancelOnArrival(t *testing.T, arrived <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	setCommandContext(t, ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case <-arrived:
			cancel()
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
			cancel()
		}
	}()
	t.Cleanup(func() { cancel(); <-done })
}

func TestEveryCLIClientEndsWithTheCommandContext(t *testing.T) {
	cases := []struct {
		name string
		call func(t *testing.T, api string) error
	}{
		{"clientFromFlag", func(t *testing.T, api string) error {
			_, err := clientFromFlag(api).Whoami()
			return err
		}},
		{"runAuthCommand", func(t *testing.T, api string) error {
			var code int
			stderr := captureStderr(t, func() {
				_, code = runAuthCommand("forgot-password", []string{"--api", api, "a@example.org"})
			})
			if code == 0 {
				return nil // the API answered: nothing cancelled the request
			}
			return errors.New(strings.TrimSpace(stderr))
		}},
		{"setupClient", func(t *testing.T, api string) error {
			sc := &setupClient{}
			sc.rebind(api)
			_, err := sc.AuthConfig()
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			t.Setenv("XDG_CONFIG_HOME", "")
			t.Setenv("INSIDEOUT_API", "")
			t.Setenv("INSIDEOUT_TOKEN", "tok-test")
			api, arrived := holdingAPI(t)
			cancelOnArrival(t, arrived)

			start := time.Now()
			err := tc.call(t, api)
			if took := time.Since(start); took > 2*time.Second {
				t.Errorf("returned after %v, want an abort shortly after cliCtx was cancelled", took)
			}
			if err == nil || !strings.Contains(err.Error(), context.Canceled.Error()) {
				t.Errorf("err = %v, want the cancelled context's error", err)
			}
		})
	}
}

// The setup form builds its client before it knows a server (the hosted API is
// the default until the form rebinds it); that client carries cliCtx too.
func TestSetupClientBeforeAnyRebindIsOnTheCommandContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	setCommandContext(t, ctx)
	if got := (&setupClient{}).client().Context(); got != ctx {
		t.Errorf("the default setup client is on %v, want the command's context", got)
	}
}
