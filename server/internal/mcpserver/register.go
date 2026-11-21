package mcpserver

// One way to register a tool, and the argument readers every tool shares
// (rv5-STRS-02). Before this file each registrar carried its own copy of the
// closure and readers, and a copy could drop the tool-call lock.

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// toolFunc is one tool's body: the client it runs on and the call's arguments.
type toolFunc func(c *apiclient.Client, req mcp.CallToolRequest) (string, error)

// registerTool adds one tool to s. Every handler holds the tool-call lock for
// its whole run (beginToolCall), builds its client from the call's context, and
// turns an error into a tool error rather than a protocol error.
func registerTool(s *server.MCPServer, name, desc string, opts []mcp.ToolOption,
	open func(context.Context) *apiclient.Client, result func(string) *mcp.CallToolResult, call toolFunc) {
	all := append([]mcp.ToolOption{mcp.WithDescription(desc)}, opts...)
	s.AddTool(mcp.NewTool(name, all...), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		defer beginToolCall()()
		out, err := call(open(ctx), req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return result(out), nil
	})
}

// toolSet is the server and API base every registrar adds its tools to.
type toolSet struct {
	s   *server.MCPServer
	api string
}

// add registers a product tool: it runs on the agent-scope token (clientCtx)
// and its result is labelled and fenced as data (textResult).
func (ts toolSet) add(name, desc string, opts []mcp.ToolOption, call toolFunc) {
	registerTool(ts.s, name, desc, opts, func(ctx context.Context) *apiclient.Client { return clientCtx(ctx, ts.api) }, textResult, call)
}

// addHuman registers an auth tool: these open, close and recover the login, so
// they run on the user's own client and their plain text is the tool's own.
func (ts toolSet) addHuman(name, desc string, opts []mcp.ToolOption, call toolFunc) {
	registerTool(ts.s, name, desc, opts, func(ctx context.Context) *apiclient.Client { return humanClientCtx(ctx, ts.api) }, mcp.NewToolResultText, call)
}

func strReq(name, desc string) mcp.ToolOption {
	return mcp.WithString(name, mcp.Required(), mcp.Description(desc))
}

func strOpt(name, desc string) mcp.ToolOption {
	return mcp.WithString(name, mcp.Description(desc))
}

// reqStr reads a required string argument; MCP validation already
// guarantees presence, so a miss degrades to "" and the API answers 4xx.
func reqStr(req mcp.CallToolRequest, name string) string {
	v, _ := req.RequireString(name)
	return v
}

// optStr reads an optional string argument; absent is "".
func optStr(req mcp.CallToolRequest, name string) string {
	v, _ := req.GetArguments()[name].(string)
	return v
}

func call(raw json.RawMessage, err error) (string, error) {
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
