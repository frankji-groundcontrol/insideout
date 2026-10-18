// Package mcpserver builds the InsideOut MCP server: one tool per CLI
// verb over the same /api/v1 contract. It lives here rather than in
// cmd/insideout-mcp so the tool registry, the in-process session, and the
// stdio protocol can be exercised end-to-end in tests.
package mcpserver

import (
	"encoding/json"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// Build returns the MCP server for the given API base. An empty base
// resolves like the CLI (INSIDEOUT_API, the saved login's api_base, then the
// hosted API) and the token comes from INSIDEOUT_TOKEN or the saved login —
// all through clientconfig.Resolve — or sign the process in at runtime with
// the auth_* tools (2026-08-24: all four surfaces must agree on auth ways; a
// login tool's token is process state, never conversation output).
func Build(api string) *server.MCPServer {
	s := server.NewMCPServer("insideout", "0.1.0",
		server.WithToolCapabilities(false),
		server.WithInstructions("InsideOut: ideas → PRDs → roadmaps. `guide` scaffolds the "+
			"repo-side insideout.yaml matching guide; roadmap_* manage branches; build/expand "+
			"run the agent planner. Auth: INSIDEOUT_TOKEN env, or the auth_* tools — "+
			"prefer auth_login_code_request/auth_login_code_verify (the human supplies the "+
			"emailed digits; no password in the conversation)."))

	addAuthTools(s, api)
	addProductTools(s, api)
	addAdminTools(s, api)
	return s
}

// session holds the token acquired by an auth tool so subsequent tool
// calls use it; the saved credentials stay the fallback. Process memory,
// never conversation output.
var session struct {
	sync.Mutex
	token string
}

func setSessionToken(tok string) {
	session.Lock()
	session.token = tok
	session.Unlock()
}

// client resolves the saved credentials like every other surface
// (clientconfig.Resolve) and lets this process's own login win over them.
// Credentials that cannot be resolved (an unreadable config file) give a
// client that fails every call with that error.
func client(api string) *apiclient.Client {
	creds, err := clientconfig.Resolve(api)
	if err != nil {
		return apiclient.Failing(err)
	}
	c := creds.Client()
	session.Lock()
	inProcess := session.token
	session.Unlock()
	if inProcess != "" {
		c.SetToken(inProcess)
	}
	return c
}

// reqStr reads a required string argument; MCP validation already
// guarantees presence, so a miss degrades to "" and the API answers 4xx.
func reqStr(req mcp.CallToolRequest, name string) string {
	v, _ := req.RequireString(name)
	return v
}

func call(raw json.RawMessage, err error) (string, error) {
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
