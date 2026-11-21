package mcpserver

import (
	"slices"
	"testing"
)

// rv5-PARA-07: every tool runs on an agent-scope token once a login is held, and
// answers 401 before one is, so a tool on a human-only route can never succeed.
// Such tools are not registered at all; signup_mode is the public read, listed
// with or without a login.
func TestNoToolIsRegisteredThatCanNeverSucceed(t *testing.T) {
	pinConfigHome(t)
	never := []string{"commit", "proposal_decide", "invite_create", "invite_list", "invite_revoke", "idea_delete", "roadmap_delete"}
	for _, gone := range never {
		if slices.Contains(offeredTools(t, Build(exampleAPI)), gone) {
			t.Errorf("%s is registered, but its route refuses every token this process can send", gone)
		}
	}

	t.Setenv("INSIDEOUT_TOKEN", humanJWT())
	for _, tool := range listTools(t, Build(exampleAPI)) {
		if tool.Name != "signup_mode" {
			continue
		}
		if _, ok := tool.InputSchema.Properties["mode"]; ok {
			t.Errorf("signup_mode takes a mode, but setting it is an admin route no MCP token reaches")
		}
		return
	}
	t.Error("signup_mode, the public read, is not offered to a process holding a login")
}

// rv5-COMP-03 follow-up: the MCP always acts through an agent-scoped token,
// and the server keeps delegation, hand-back and the member list for a human.
// Those tools could never succeed: signed in they are refused, and before a
// sign-in they fail for lack of a login. So they are not registered at all.
func TestTheMCPDoesNotRegisterToolsAnAgentTokenCanNeverUse(t *testing.T) {
	pinConfigHome(t)
	offered := offeredTools(t, Build(exampleAPI))
	for _, name := range []string{"members", "roadmap_delegate", "roadmap_handback"} {
		if slices.Contains(offered, name) {
			t.Errorf("a process with no login offers %s, which an agent token can never use", name)
		}
	}
}
