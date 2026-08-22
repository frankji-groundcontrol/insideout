# Usage Docs

How to run InsideOut — as a developer or an operator.

| Doc | Audience | Covers |
|-----|----------|--------|
| [local-development.md](local-development.md) | Developers | Prerequisites, env setup, database setup, running server + frontend, tests |
| [../client/README.md](../../client/README.md) | Developers | Flutter client (`client/`): web / iOS / Android against the Go API |
| [environment.md](environment.md) | Everyone | Environment variable reference: required/optional vars, defaults, the .env→process bridges (dev.sh, docker-compose), offline AI mode |
| [../SETENV.md](../SETENV.md) | First-timers | The environment operating manual: create the file, choose each key (`env.sh init` / `edit`), propagate to the components, prove it works |
| [deployment.md](deployment.md) | Operators | docker-compose topology, image builds, ports, reverse-proxy expectations, current Railway public deploy |
| [cli-mcp-and-guidance.md](cli-mcp-and-guidance.md) | Users + agents | The `insideout` CLI, the MCP server registration, and the `insideout.yaml` guidance flow that feeds GitHub evidence into a roadmap |

For what the product *is*, see the root [README.md](../../README.md).

> **Note:** [local-development.md](local-development.md) supersedes the old
> `docs/INSTALL.md` (removed 2026-07-21; see git history), which predated the
> Go rewrite and described a toolchain setup that is no longer how this repo
> is developed.
