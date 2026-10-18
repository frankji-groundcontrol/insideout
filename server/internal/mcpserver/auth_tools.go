package mcpserver

// The auth tools — the MCP projection of every way the web can sign in,
// reversing the earlier no-login-tool stance by the user's product call
// (all four surfaces must agree). A successful login stores the token in
// process memory shared by every tool call; the token itself is never
// tool output, so secrets stay out of the conversation. The code flow is
// the agent-friendly one: the human receives the email and supplies the
// digits — no password in the transcript.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

func addAuthTools(s *server.MCPServer, api string) {
	strReq := func(name, desc string) mcp.ToolOption {
		return mcp.WithString(name, mcp.Required(), mcp.Description(desc))
	}
	strOpt := func(name, desc string) mcp.ToolOption {
		return mcp.WithString(name, mcp.Description(desc))
	}

	// signedIn confirms the freshly stored session via /me and reports the
	// user — proof the login worked without echoing the token.
	signedIn := func(ctx context.Context) (string, error) {
		applyAgentProfile(s, api) // a process holding a login does not offer the human-only tools
		me, err := clientCtx(ctx, api).Whoami()
		if err != nil {
			return "", err
		}
		var u struct {
			Email    string `json:"email"`
			Username string `json:"username"`
		}
		_ = json.Unmarshal(me, &u)
		return fmt.Sprintf("signed in as %s (%s) — the session is active for this MCP server process; the token is not echoed. Call whoami any time.", u.Email, u.Username), nil
	}

	addTool := func(name, desc string, opts []mcp.ToolOption, call func(c *apiclient.Client, req mcp.CallToolRequest) (string, error)) {
		all := append([]mcp.ToolOption{mcp.WithDescription(desc)}, opts...)
		s.AddTool(mcp.NewTool(name, all...), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// The user's own client: these tools open, close and recover the
			// login, so none of them runs on an agent-scope token.
			out, err := call(humanClientCtx(ctx, api), req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(out), nil
		})
	}

	addTool("auth_register", "Create an account and sign this MCP server process in (password appears in the conversation — prefer auth_login_code_request for humans-in-the-loop)",
		[]mcp.ToolOption{strReq("email", "account email"), strReq("password", "password (min 8 chars)"), strReq("username", "display name"), strOpt("invite_code", "invitation code, required when signup is invitation-only")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			invite, _ := req.GetArguments()["invite_code"].(string)
			if err := c.Register(reqStr(req, "email"), reqStr(req, "password"), reqStr(req, "username"), invite); err != nil {
				return "", err
			}
			setSessionToken(c.Token())
			return signedIn(c.Context())
		})
	addTool("auth_login", "Password login for this MCP server process (password appears in the conversation — prefer auth_login_code_request for humans-in-the-loop)",
		[]mcp.ToolOption{strReq("email", "account email"), strReq("password", "account password")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			if err := c.Login(reqStr(req, "email"), reqStr(req, "password")); err != nil {
				return "", err
			}
			setSessionToken(c.Token())
			return signedIn(c.Context())
		})
	addTool("auth_login_code_request", "Email a 6-digit sign-in code to the account (uniform ok — never reveals whether it exists). Then ask the human for the digits and call auth_login_code_verify",
		[]mcp.ToolOption{strReq("email", "account email")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			if err := c.RequestLoginCode(reqStr(req, "email")); err != nil {
				return "", err
			}
			return "code sent if the account exists — ask the human for the 6 digits, then call auth_login_code_verify", nil
		})
	addTool("auth_login_code_verify", "Verify the emailed 6-digit code and sign this MCP server process in",
		[]mcp.ToolOption{strReq("email", "account email"), strReq("code", "6-digit code from the email")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			if err := c.VerifyLoginCode(reqStr(req, "email"), reqStr(req, "code")); err != nil {
				return "", err
			}
			setSessionToken(c.Token())
			return signedIn(c.Context())
		})
	addTool("auth_magic_link_request", "Email a one-time sign-in link (uniform ok). Then ask the human for the link (or just the token after token=) and call auth_magic_link_consume",
		[]mcp.ToolOption{strReq("email", "account email")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			if err := c.RequestMagicLink(reqStr(req, "email")); err != nil {
				return "", err
			}
			return "link sent if the account exists — ask the human for it, then call auth_magic_link_consume with the token (the part after token=)", nil
		})
	addTool("auth_magic_link_consume", "Exchange an emailed magic-link token for a session for this MCP server process",
		[]mcp.ToolOption{strReq("token", "token from the emailed link (the part after token=)")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			if err := c.ConsumeMagicLink(reqStr(req, "token")); err != nil {
				return "", err
			}
			setSessionToken(c.Token())
			return signedIn(c.Context())
		})
	addTool("auth_forgot_password", "Email a one-time password-reset link (uniform ok — never reveals whether it exists)",
		[]mcp.ToolOption{strReq("email", "account email")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			if err := c.ForgotPassword(reqStr(req, "email")); err != nil {
				return "", err
			}
			return "reset link sent if the account exists — the human can finish with auth_reset_password (token from the email) or in the browser", nil
		})
	addTool("auth_reset_password", "Set a new password from an emailed reset token; every session of the account is revoked, so sign in again afterwards",
		[]mcp.ToolOption{strReq("token", "token from the emailed reset link"), strReq("password", "new password (min 8 chars)")},
		func(c *apiclient.Client, req mcp.CallToolRequest) (string, error) {
			if err := c.ResetPassword(reqStr(req, "token"), reqStr(req, "password")); err != nil {
				return "", err
			}
			return "password reset and all sessions revoked — sign in again (auth_login, or the code flow)", nil
		})
	addTool("auth_logout", "Clear this MCP server process's in-process session (INSIDEOUT_TOKEN env, if set, becomes the credential again)",
		nil,
		func(c *apiclient.Client, _ mcp.CallToolRequest) (string, error) {
			setSessionToken("")
			return "session cleared", nil
		})
}
