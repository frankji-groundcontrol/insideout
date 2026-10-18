# Usage Docs

How to run InsideOut — as a developer or an operator.

| Doc | Audience | Covers |
|-----|----------|--------|
| [installation.md](installation.md) | Users + agents | Hosted install with no environment variables, and the guided PostgreSQL setup (`insideout setup`) for a server on this machine |
| [key-setup-tutorial.md](key-setup-tutorial.md) | Operators + first-timers | Guided setup for all 31 mapped keys, with links to each key reference |
| [key-testing-tutorial.md](key-testing-tutorial.md) | Operators + maintainers | Shape, reachability, provider, delivery, and capability checks for every key |
| [../../skills/insideout-setup/](../../skills/insideout-setup/SKILL.md) | Agents | The installation skill: bare macOS/Ubuntu machine → verified install, module-by-module steps, key store, host record |
| [local-development.md](local-development.md) | Developers | Prerequisites, env setup, database setup, running server + frontend, tests |
| [../client/README.md](../../client/README.md) | Developers | Flutter client (`client/`): web / iOS / Android against the Go API |
| [environment.md](environment.md) | Everyone | Environment variable reference: required/optional vars, defaults, the .env→process bridges (dev.sh, docker-compose), offline AI mode |
| [keys/](keys/README.md) | Operators | The host key store: one standalone six-part file per credential kind (where / propagated / set / tested / used / install guide), index + shared flow + two-layer testing (shape suite + live probes) + per-host state |
| [../SETENV.md](../SETENV.md) | First-timers | The environment operating manual: create the file, choose each key (`env.sh init` / `edit`), propagate to the components, prove it works |
| [deployment.md](deployment.md) | Operators | docker-compose topology, image builds, ports, reverse-proxy expectations, current Railway public deploy |
| [backup-restore.md](backup-restore.md) | Operators | The backup story: managed (Supabase toggle) + portable dumps (`scripts/backup-db.sh`), restore runbook, scratch-stack walkthrough |
| [releasing.md](releasing.md) | Maintainers | How a `v*` tag becomes a release: the test step, the four builds and checksums, the publish, and the decisions still open (version policy, rollback, a stronger gate) |
| [cli-mcp-and-guidance.md](cli-mcp-and-guidance.md) | Users + agents | The `insideout` CLI (`scripts/install.sh` → `~/.local/bin` + PATH), the MCP server registration (`scripts/mcp-config.sh`), and the `insideout.yaml` guidance flow that feeds GitHub evidence into a roadmap |

For what the product *is*, see the root [README.md](../../README.md).

> **Note:** [local-development.md](local-development.md) supersedes the old
> `docs/INSTALL.md` (removed 2026-07-21; see git history), which predated the
> Go rewrite and described a toolchain setup that is no longer how this repo
> is developed.
