package mcpserver

import (
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// Since AIDEV-04 the registry and the token follow the credential in the
// environment: a JWT in INSIDEOUT_TOKEN, or a saved login, withdraws the
// human-only tools and sends every tool call through the agent-token exchange,
// which the fake APIs these tests build servers against do not serve. A test
// must not inherit the credential of the machine running it, so main_test.go
// starts the package from an empty home with no token. Run the package with a
// JWT exported as INSIDEOUT_TOKEN to see what this guards.
func TestThePackageStartsWithoutTheDevelopersCredential(t *testing.T) {
	creds, err := clientconfig.Resolve(exampleAPI)
	if err != nil {
		t.Fatalf("resolving the credentials of an empty home: %v", err)
	}
	if creds.Token != "" {
		t.Errorf("the package starts holding a %d-byte credential it did not set: INSIDEOUT_TOKEN or a saved login leaks in from the machine, and the tests built on Build see a narrowed registry and a failed token exchange", len(creds.Token))
	}
}
