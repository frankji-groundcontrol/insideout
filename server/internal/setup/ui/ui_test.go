package ui

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/setup/flow"
)

// Under `go test` the test binary's stdout is a pipe (the go tool
// captures it), so the ErrNoTTY branch is directly reachable. If some
// future harness hands us a real terminal on both ends, skip rather
// than hang on an interactive form.
func stdinAndStdoutAreCharacterDevices(t *testing.T) bool {
	t.Helper()
	charDevice := func(f *os.File) bool {
		info, err := f.Stat()
		if err != nil {
			return false
		}
		return info.Mode()&os.ModeCharDevice != 0
	}
	if charDevice(os.Stdin) && charDevice(os.Stdout) {
		t.Log("stdin and stdout are both character devices; skipping to avoid an interactive hang")
		return true
	}
	return false
}

func TestRunRefusesWithoutTTY(t *testing.T) {
	if stdinAndStdoutAreCharacterDevices(t) {
		t.Skip("interactive terminal attached; ErrNoTTY path unreachable here")
	}
	_, err := Run(context.Background(), flow.Deps{DefaultBaseURL: "https://default.test/api/v1"})
	var noTTY ErrNoTTY
	if !errors.As(err, &noTTY) {
		t.Fatalf("Run() error = %v, want ErrNoTTY", err)
	}
	msg := noTTY.Error()
	for _, want := range []string{
		"interactive setup needs a terminal",
		"insideout setup --hosted",
		"--local --postgres bundled|own",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("ErrNoTTY message %q is missing %q", msg, want)
		}
	}
}
