// Package mcpserver builds the InsideOut MCP server: one tool per CLI
// verb over the same /api/v1 contract. It lives here rather than in
// cmd/insideout-mcp so the tool registry, the in-process session, and the
// stdio protocol can be exercised end-to-end in tests.
package mcpserver

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// Build returns the MCP server for the given API base. Configure with
// INSIDEOUT_API (default: hosted API) and INSIDEOUT_TOKEN, or sign the
// process in at runtime with the auth_* tools (2026-08-24: all four
// surfaces must agree on auth ways; a login tool's token is process
// state, never conversation output).
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
// calls use it; INSIDEOUT_TOKEN stays the fallback. Process memory, never
// conversation output.
var session struct {
	sync.Mutex
	token string
}

// sessionToken returns the in-process login token, falling back to
// INSIDEOUT_TOKEN.
func sessionToken() string {
	session.Lock()
	defer session.Unlock()
	if session.token != "" {
		return session.token
	}
	return os.Getenv("INSIDEOUT_TOKEN")
}

func setSessionToken(tok string) {
	session.Lock()
	session.token = tok
	session.Unlock()
}

func client(api string) *apiclient.Client {
	c := apiclient.New(api)
	c.SetToken(sessionToken())
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
