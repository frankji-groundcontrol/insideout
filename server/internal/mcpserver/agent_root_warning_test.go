package mcpserver

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// buildStderr builds the server and returns what Build wrote to stderr.
func buildStderr(t *testing.T) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	Build(exampleAPI)
	os.Stderr = old
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

// INSIDEOUT_AGENT_ROOT asks for a root-scoped token when the process trades a
// login. A token that already carries a scope is sent as it is, so the root
// would be silently ignored: the process says so once, at startup.
func TestAnAgentRootBesideAScopedTokenIsWarnedAboutAtStartup(t *testing.T) {
	pinConfigHome(t)
	t.Setenv("INSIDEOUT_AGENT_ROOT", "4f0d8c5e-5d3b-4f6e-9a55-0d9a2f1c7b22")
	t.Setenv("INSIDEOUT_TOKEN", fakeJWT("agent", "scoped", time.Now(), 15*time.Minute))
	if got := buildStderr(t); strings.Count(got, "\n") != 1 || !strings.Contains(got, "INSIDEOUT_AGENT_ROOT") {
		t.Errorf("scoped token with a root: stderr %q, want one warning line naming INSIDEOUT_AGENT_ROOT", got)
	}
	t.Setenv("INSIDEOUT_TOKEN", humanJWT())
	if got := buildStderr(t); got != "" {
		t.Errorf("a login the process trades with the root: stderr %q, want nothing", got)
	}
}
