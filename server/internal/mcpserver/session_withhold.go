package mcpserver

import (
	"context"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// sessionSwapTools are how a human signs this process in, or clears it.
// They stay advertised until a login is held. After that a prompt must not
// be offered a way to replace the identity: restart the process to change
// it. The handlers stay registered, so a call still hits refuseSessionSwap.
var sessionSwapTools = []string{
	"auth_login_code_verify",
	"auth_magic_link_consume",
	"auth_logout",
}

// withholdSessionSwapTools drops sessionSwapTools from tools/list while this
// process holds a JWT (saved login, INSIDEOUT_TOKEN, or an in-process
// sign-in). auth_login_code_request and auth_magic_link_request stay.
func withholdSessionSwapTools(api string) server.ToolFilterFunc {
	return func(_ context.Context, tools []mcp.Tool) []mcp.Tool {
		if _, jwt := apiclient.ParseTokenClaims(humanClient(api).Token()); !jwt {
			return tools
		}
		kept := make([]mcp.Tool, 0, len(tools))
		for _, tool := range tools {
			if slices.Contains(sessionSwapTools, tool.Name) {
				continue
			}
			kept = append(kept, tool)
		}
		return kept
	}
}
