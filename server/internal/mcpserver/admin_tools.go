package mcpserver

// The admin tools — invitation revocation and the signup-mode switch,
// the MCP projection of `invite revoke` and `signup-mode`. signup_mode
// with no mode argument reads the public /auth/config, so an agent can
// learn whether registration needs an invitation code before signing up.

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

func addAdminTools(s *server.MCPServer, api string) {
	strReq := func(name, desc string) mcp.ToolOption {
		return mcp.WithString(name, mcp.Required(), mcp.Description(desc))
	}
	strOpt := func(name, desc string) mcp.ToolOption {
		return mcp.WithString(name, mcp.Description(desc))
	}

	addTool := func(name, desc string, opts []mcp.ToolOption, call func(c *apiclient.Client, req mcp.CallToolRequest) (string, error)) {
		all := append([]mcp.ToolOption{mcp.WithDescription(desc)}, opts...)
		s.AddTool(mcp.NewTool(name, all...), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			out, err := call(clientCtx(ctx, api), req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return textResult(out), nil
		})
	}

	addTool("invite_revoke", "Admin: revoke a still-usable invitation (the code stops working)",
		[]mcp.ToolOption{strReq("invitation_id", "invitation id from invite_list")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			if err := c.RevokeInvitation(reqStr(req, "invitation_id")); err != nil {
				return "", err
			}
			return "revoked", nil
		})
	addTool("signup_mode", "Read the signup mode (no mode argument — public), or set it (admin): invitation_only|open|closed",
		[]mcp.ToolOption{strOpt("mode", "invitation_only|open|closed; omit to read the current mode")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			mode, _ := req.GetArguments()["mode"].(string)
			if mode == "" {
				return c.AuthConfig()
			}
			if mode != "invitation_only" && mode != "open" && mode != "closed" {
				return "", fmt.Errorf("mode must be invitation_only, open, or closed")
			}
			return call(c.SetSignupMode(mode))
		})
}
