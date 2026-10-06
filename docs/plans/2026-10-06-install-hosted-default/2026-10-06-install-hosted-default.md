# 2026-10-06 — Install reaches the hosted API

A finished install must be usable. The CLI and MCP server call the
hosted API when `INSIDEOUT_API` is unset. A server on this machine is
`scripts/setup.sh`, documented in
[installation.md](../../usage/installation.md).

## Outcome

`insideout whoami` with a clean environment reaches the hosted API
and returns unauthorized until the user signs in. Local servers opt
in with `INSIDEOUT_API`.

## Checklist

- [x] One default API shared by the CLI, MCP server, and `mcp-config.sh`.
- [x] Installer prints that URL and does not require environment variables.
- [x] `scripts/setup.sh` guides hosted confirm or local Postgres.
- [x] [installation.md](../../usage/installation.md) covers both paths
      and the questions an agent asks.
- [x] `insideout whoami` against the hosted API returns unauthorized,
      not connection refused.

## Status

Finished in the worktree 2026-10-06. Not deployed. Re-run
`./scripts/install.sh` on machines that already have the old binary.

Records:
[changelog](../../changelogs/2026-10-06-install-hosted-default.md),
[learning](../../learning/2026-10-06-localhost-default-makes-install-unusable.md).
