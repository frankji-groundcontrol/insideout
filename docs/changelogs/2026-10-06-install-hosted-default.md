# 2026-10-06 — Install calls the hosted API

A finished `./scripts/install.sh` left the CLI and MCP server aimed at
`127.0.0.1:8080`, so `insideout whoami` died with connection refused.
The hosted API is what a new install calls now. Nothing was deployed.

## What changed

- `server/internal/apiclient/default_base.go` holds the one default API.
  The CLI and `insideout-mcp` use it when `INSIDEOUT_API` is unset.
- `scripts/default-api.sh` prints that URL. `scripts/mcp-config.sh`
  uses it and omits `INSIDEOUT_TOKEN` when none is set, so a placeholder
  token is not sent.
- `scripts/install.sh` prints the URL and how to sign in. On a terminal
  it offers `scripts/setup.sh`.
- `scripts/setup.sh` is the guided local-server path: hosted confirm,
  bundled Docker Postgres, or an existing database URL on stdin. It
  writes `.env` without printing values, then starts Postgres and
  migrates unless `--no-start`.
- [installation.md](../usage/installation.md) is the user and agent
  walk-through. README, SKILL, and the CLI usage doc point at it.

Local servers still opt in with
`INSIDEOUT_API=http://127.0.0.1:8080/api/v1`.

## Verification

`go test ./internal/apiclient/ -run 'TestDefaultBase|TestMCPConfig'`
passed. `bash -n` on the four scripts. `setup.sh --hosted` prints the
hosted path. `setup.sh --local --postgres bundled --no-start` against a
temp env file wrote a localhost `DATABASE_URL` and a generated JWT and
did not print them. After reinstall, `insideout whoami` with no token
and no `INSIDEOUT_API` reached the hosted API and returned unauthorized
instead of connection refused.

## Operator

Nothing to migrate. Anyone who already installed the CLI should
re-run `./scripts/install.sh` so the binary picks up the new default.
Set `INSIDEOUT_API` only for a server on that machine.
