package mcpserver

import (
	"fmt"
	"os"
	"testing"
)

// TestMain starts every test of the package from an empty home with no token
// and no API base, so the machine's own login cannot reach them (see
// ambient_env_test.go). A test that needs a credential sets its own, as
// pinConfigHome and t.Setenv do, and the value it set is gone when it ends.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "mcpserver-test-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "mcpserver tests:", err)
		os.Exit(1)
	}
	for k, v := range map[string]string{
		"HOME":            home,
		"XDG_CONFIG_HOME": "",
		"INSIDEOUT_API":   "",
		"INSIDEOUT_TOKEN": "",
	} {
		os.Setenv(k, v)
	}
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}
