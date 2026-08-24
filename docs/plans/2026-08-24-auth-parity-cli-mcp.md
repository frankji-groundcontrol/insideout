# 2026-08-24 — Auth parity: CLI + MCP get every login/register way

Status: **Finished** — all checklist items verified; see the
[changelog](../changelogs/2026-08-24-auth-parity-cli-mcp.md) for the
verification evidence.

## Why this exists

The email-auth slice gave the API and the web all five auth flows
(password login, register, magic link, password reset, code login). The
user then required surface agreement: **web, CLI, API, MCP must all allow
login/register in the different ways**. Today the CLI has password login
only (`insideout login <email>`, token on stdout) and the MCP server has
no login at all — `INSIDEOUT_TOKEN` env, with a deliberately-absent login
tool ("token is environment state, not conversation state",
docs/usage/cli-mcp-and-guidance.md). The user's product call reverses
that stance: MCP gets auth tools too.

## Design

- `internal/apiclient` (shared CLI/MCP client) gains the auth verbs,
  one per API route: `Register`, `RequestMagicLink`, `ConsumeMagicLink`,
  `RequestLoginCode`, `VerifyLoginCode`, `ForgotPassword`,
  `ResetPassword`. Session-shaped calls store the access token exactly
  like `Login` (refresh stays unused — CLI/MCP log in again on expiry).
- CLI (new `cmd/insideout/cli_auth.go`, prompts on stderr, token-only
  stdout — the existing convention):
  - `insideout register <email> [display-name]` — password via stdin.
  - `insideout login <email>` — unchanged password flow (default).
  - `insideout login --code <email>` — emails a 6-digit code, digits read
    from stdin, verifies, prints token.
  - `insideout login --magic <email>` — emails the link; the user pastes
    the link or raw token (`tokenFromLink` helper extracts it); consumes,
    prints token.
  - `insideout forgot-password <email>` — triggers the uniform-200 email.
  - `insideout reset-password` — pasted link/token + new password on
    stdin; the API revokes all sessions.
- MCP (`cmd/insideout-mcp`): 8 tools — `auth_register`, `auth_login`,
  `auth_login_code_request`, `auth_login_code_verify`,
  `auth_magic_link_request`, `auth_magic_link_consume`,
  `auth_forgot_password`, `auth_reset_password`. A process-wide session
  holder replaces the per-call client: a successful login tool stores the
  token in-process and every subsequent call uses it (`INSIDEOUT_TOKEN`
  stays the fallback). Tool output confirms "signed in as …" and never
  echoes the token — secrets stay out of the conversation; the code flow
  is the agent-friendly one (the human supplies the emailed code, no
  password in the transcript). Tool descriptions say this.

## Out of scope

- Persisted CLI credential files / refresh-token rotation for CLI/MCP
  (they re-login on expiry today; unchanged).
- Web client (already complete); API surface (already complete).

## Engineering review

Self-review (additive, pattern-matched to the existing apiclient/CLI/MCP
layers): no new trust surface — the tools wrap existing public routes;
the MCP session is per-process memory, not conversation state; wrong
credentials surface as tool errors from the API's error contract. The
one deliberate reversal (MCP login tools vs. the old no-login-tool
stance) is the user's explicit product decision, recorded here and in
the changelog.

## Checklist

- [x] Plan + board row
- [x] apiclient auth methods, tests first (httptest)
- [x] CLI verbs + `tokenFromLink` helper test
- [x] MCP session holder + 8 auth tools (+ logout → 9; server refactored
      into `internal/mcpserver` for stdio-level tests)
- [x] `go build/vet/test` green
- [x] Live exercise against a local server: CLI register/login --code/
      login --magic/forgot/reset; MCP JSON-RPC code login + whoami
- [x] Usage doc + changelog + board/plan closure
- [ ] Commit, push, deploy server (binaries ship in the image)
