package mcpserver

import (
	"strings"
	"testing"
)

// R1-03: create-invitation answers with mailSent (rv4-USER-07), and false means
// nothing was e-mailed, so the admin has to hand the signup URL over. A model
// that learns the tool only from the tools/list description has to find that
// there, or it reports an invitation as sent when no mail went out. The test
// reads the description a client receives on the wire, from a process with no
// credential (main_test.go), which still offers the whole registry.
func TestInviteCreateDescriptionNamesMailSent(t *testing.T) {
	desc, ok := toolDescriptions(t)["invite_create"]
	if !ok {
		t.Fatal("invite_create is not offered to a process with no credential")
	}
	for _, want := range []string{"mailSent", "false", "nothing was e-mailed"} {
		if !strings.Contains(desc, want) {
			t.Errorf("the invite_create description lacks %q, so a model cannot tell an invitation that went out from one that did not: %q", want, desc)
		}
	}
}
