package mcpserver

// The product tools — one per CLI verb, names/arguments/output 1:1 with
// the CLI (docs/plans/2026-08-21-cli-mcp-parity.md). Reads return the
// API's JSON, not a reshaping of it, so surfaces cannot drift from the truth;
// textResult labels and fences it (fence.go), since most of it is text other
// people wrote. Each resource registers its tools from its own tools_*.go file.

import (
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

func addProductTools(ts toolSet) {
	addAccountTools(ts)
	addWorkspaceTools(ts)
	addProjectTools(ts)
	addProjectUpdateTools(ts)
	addIdeaTools(ts)
	addPrdTools(ts)
	addPrdHistoryTools(ts)
	addRoadmapTools(ts)
	addDraftTools(ts)
	addRoadmapTreeTools(ts)
	addRepoTools(ts)
	addAgentTools(ts)
}

// addAccountTools registers who the caller is and the public half of the admin
// verbs: whether signing up needs an invitation. Setting the mode and the
// invitations are /admin/* routes, kept for a human.
func addAccountTools(ts toolSet) {
	ts.add("whoami", "GET /me — the authenticated user", nil,
		func(c *apiclient.Client, _ mcp.CallToolRequest) (string, error) { return call(c.Whoami()) })
	ts.add("signup_mode", "Read the signup mode (public, no sign-in needed): invitation_only|open|closed", nil,
		func(c *apiclient.Client, _ mcp.CallToolRequest) (string, error) { return c.AuthConfig() })
}
