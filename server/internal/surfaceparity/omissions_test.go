package surfaceparity

// omission is one request field a surface deliberately does not reach.
type omission struct {
	route     string // the mux pattern, as registered in internal/api
	key       string // the JSON key, or "*" for every key of the route
	surfaces  string // comma-separated: client, cli, mcp
	humanOnly bool   // the route is humanOnly in agent_scope.go's agentRoutePolicy
	reason    string
}

// omissions is the one table of deliberate gaps (docs/usage/cli-mcp-and-guidance.md,
// "MCP registers no tool for a human-only route"). An entry marked humanOnly
// takes its reason from the comment over the route in agent_scope.go; the test
// fails when that file and this table disagree. There is no table of
// accidental gaps: the walk found one (profile keywords on the CLI) and it was
// wired.
var omissions = []omission{
	// Decisions stay with the signed-in human.
	{"POST /api/v1/agent/proposals/{uid}/decision", "*", "mcp", true, "a decision under the proposer's own id would be the author deciding its own proposal"},
	{"POST /api/v1/prds/{id}/commit", "*", "mcp", true, "Commit is a human's decision"},
	{"POST /api/v1/agent/token", "*", "client,cli,mcp", true, "an agent token never mints another; the client trades for one itself (AgentScopedTo), no verb sets rootNodeId"},

	// The instance: signup mode, invitations, the mail probe.
	{"PUT /api/v1/admin/settings", "*", "mcp", true, "the instance switches are an admin's"},
	{"POST /api/v1/admin/invitations", "*", "mcp", true, "the instance switches are an admin's"},
	{"POST /api/v1/admin/mail/probe", "*", "client,cli,mcp", true, "an operator's route, called by hand (docs/usage/keys/mail/README.md)"},

	// Every delete.
	{"DELETE /api/v1/me", "*", "mcp", true, "every delete is a human's; account deletion asks for the password"},

	// No agent tool calls these: the coach conversation (the web client's),
	// edits to people and members, live streams, evidence entry.
	{"POST /api/v1/conversations/{id}/messages", "*", "client,cli,mcp", true, "the coach conversation is the web client's"},
	{"POST /api/v1/conversations/{id}/stage", "*", "client,cli,mcp", true, "the coach conversation is the web client's"},
	{"POST /api/v1/conversations/{id}/stage/decline", "*", "client,cli,mcp", true, "the coach conversation is the web client's"},
	{"POST /api/v1/prds/{id}/conversation", "*", "client,cli,mcp", true, "the coach conversation is the web client's"},
	{"POST /api/v1/projects/{pid}/cursor", "*", "client,cli,mcp", true, "a live canvas cursor, the web client's"},
	{"PATCH /api/v1/me", "*", "mcp", true, "edits to people are a human's"},
	{"PATCH /api/v1/workspaces/{id}/members/{userId}", "*", "mcp", true, "edits to members are a human's"},
	{"POST /api/v1/roadmap/{nid}/evidence", "*", "mcp", true, "evidence entry is a human's observation"},
	{"POST /api/v1/roadmap/{nid}/validate", "*", "mcp", true, "validation is a human's judgement"},

	// Ownership is write authority: a human decision (owner decision 4).
	{"POST /api/v1/roadmap/{nid}/delegate", "*", "mcp", true, "ownership is write authority: a human decision"},
	{"POST /api/v1/roadmap/{nid}/handback", "*", "mcp", true, "ownership is write authority: a human decision"},

	// Fields an agent-allowed route refuses to an agent token.
	{"PATCH /api/v1/roadmap/{nid}", "ownerId", "mcp", false, "the server refuses an agent an owner change; an agent proposes one"},
	{"PATCH /api/v1/roadmap/{nid}", "completionReason", "mcp", false, "the server refuses an agent a completion reason; done rides evidence"},
	{"PATCH /api/v1/projects/{pid}", "ownerId", "mcp", false, "handleUpdateProject refuses an agent an ownerId"},

	// Session plumbing no verb sets.
	{"POST /api/v1/auth/refresh", "*", "client,cli,mcp", false, "the client renews a session by itself on a 401"},
	{"POST /api/v1/auth/logout", "*", "mcp", false, "auth_logout clears the MCP process's own session; revoking a saved login is the CLI's logout"},
}
