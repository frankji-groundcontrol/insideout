# 2026-08-24 — Auth parity: CLI + MCP get every login/register way

The user required surface agreement: **web, CLI, API, and MCP must all
allow login/register in the different ways**. The API and web already had
all five flows from the
[email-auth slice](2026-08-24-email-auth.md); the CLI had password login
only, and the MCP server had none (a deliberate no-login-tool stance).
This slice completes the matrix and reverses that stance by the user's
product call — an MCP login tool's token is process state, never
conversation output.

## What changed

- `internal/apiclient` (the shared CLI/MCP client): one method per public
  auth route — `Register`, `RequestMagicLink`, `ConsumeMagicLink`,
  `RequestLoginCode`, `VerifyLoginCode`, `ForgotPassword`,
  `ResetPassword`. Session-shaped calls store the access token exactly
  like `Login`; auth-endpoint 401s surface the API's own error text
  ("invalid or expired code") instead of the `INSIDEOUT_TOKEN` hint that
  only makes sense for authenticated calls.
- CLI (new `cmd/insideout/cli_auth.go`; prompts on stderr, token-only
  stdout): `register <email> [display-name]` (name defaults to the email
  local part), `login --code` (emails digits, reads them from stdin),
  `login --magic` (paste the emailed link or raw token — `tokenFromLink`
  accepts both), `forgot-password`, `reset-password`. The login sub-modes
  and every prompt share one buffered stdin reader (a per-prompt reader
  swallows the second answer — caught live during verification).
- MCP: `cmd/insideout-mcp` refactored into `internal/mcpserver`
  (registry + in-process session, testable) with a thin stdio main, plus
  9 auth tools (register/login/code request+verify/magic request+
  consume/forgot/reset/logout, 36 tools total). A successful login stores
  the token in process memory; `INSIDEOUT_TOKEN` stays the fallback; tool
  output reports "signed in as …" and never the token. Tool descriptions
  steer agents to the code flow (human supplies the emailed digits — no
  password in a transcript).

## Verification

- TDD: apiclient auth verbs tested against httptest first (paths, bodies,
  token capture, no bearer on pre-auth calls, API-error surfacing);
  `tokenFromLink` unit-tested; new `internal/mcpserver` integration test
  drives the **real stdio protocol** (JSON-RPC lines through
  `NewStdioServer.Listen`) against an httptest API: code request →
  verify → `whoami` rides the in-process session, logout clears it, a
  cold process has no session. `go build/vet/test` green.
- Live (mail dev mode; throwaway users deleted after, tokens never
  printed): against the hosted API — CLI `register`, password `login`,
  token-authenticated `whoami`. Against a local server on the real DB —
  CLI `login --code` (code read from the dev-mode mail log) and
  `login --magic` (full URL pasted) both issue tokens whose `whoami`
  confirms the identity; `forgot-password` → `reset-password` rotates the
  password (old password then 401s, new one signs in). Live MCP over
  stdio (`go run`): `auth_login` → "signed in as …" without echoing the
  token, then `whoami` returns the user on the in-process session.
- Environment note recorded for the record: this machine's sandbox
  blocked exec of freshly built binaries for part of the evening (new
  `/tmp` binaries hung; previously-run ones and `go run` children were
  fine) — verification went through `go run` and in-test stdio instead.

## Not done here

- Persisted CLI credentials / refresh-token rotation for CLI/MCP sessions
  (unchanged contract: re-login when the access token expires).
- Production users upgrading the installed `insideout` / `insideout-mcp`
  binaries — they must rebuild from `main` (the binaries ship in the
  server image; `railway up --service server` refreshes it).

Plan: [2026-08-24 auth parity](../plans/2026-08-24-auth-parity-cli-mcp.md).
