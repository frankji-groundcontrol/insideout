# Usage Docs

How to run InsideOut — as a developer or an operator.

| Doc | Audience | Covers |
|-----|----------|--------|
| [local-development.md](local-development.md) | Developers | Prerequisites, env setup, database setup, running server + frontend, tests |
| [environment.md](environment.md) | Everyone | Environment variable reference: required/optional vars, defaults, the .env→process bridges (dev.sh, docker-compose), offline AI mode |
| [../SETENV.md](../SETENV.md) | First-timers | The environment operating manual: create the file, choose each key (`env.sh init` / `edit`), propagate to the components, prove it works |
| [deployment.md](deployment.md) | Operators | docker-compose topology, image builds, ports, reverse-proxy expectations |

For what the product *is*, see the root [README.md](../../README.md).

> **Note:** [local-development.md](local-development.md) supersedes the old
> `docs/INSTALL.md` (removed 2026-07-21; see git history), which predated the
> Go rewrite and described a toolchain setup that is no longer how this repo
> is developed.
