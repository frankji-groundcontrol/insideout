# Usage Docs

How to run InsideOut — as a developer or an operator.

| Doc | Audience | Covers |
|-----|----------|--------|
| [user-guide.md](./user-guide.md) | Users | Prebuilt binaries, `insideout setup`, sign-in, MCP. No server variables |
| [setup-paths.md](./setup-paths.md) | Users + agents | The maintained map of every `insideout setup` path |
| [installation.md](./installation.md) | Users + agents | The full install reference: the one-liner, the hosted path, and a server on this machine |
| [../../skills/insideout-setup/](../../skills/insideout-setup/SKILL.md) | Agents | Hosted install: binaries, numbered `insideout setup`, MCP |
| [../../skills/insideout-server/](../../skills/insideout-server/SKILL.md) | Agents | Own server and client development, only when someone asked to run the server |
| [key-setup-tutorial.md](./key-setup-tutorial.md) | Developers of this repository | Guided setup for the mapped keys, with links to each key reference |
| [key-testing-tutorial.md](./key-testing-tutorial.md) | Developers of this repository | Shape, reachability, provider, delivery, and capability checks for every key |
| [local-development.md](./local-development.md) | Developers | Prerequisites, env setup, database setup, running server + frontend, tests |
| [account-deletion.md](./account-deletion.md) | Operators | How a signed-in person deletes their own account, what is removed, and what other members keep |
| [../client/README.md](../../client/README.md) | Developers | Flutter client (`client/`): web / iOS / Android against the Go API |
| [demo-data.md](./demo-data.md) | Developers | Seed a fresh local database with the Spark dating-app story through the API, and its logins |
| [environment.md](./environment.md) | Everyone | Environment variable reference: required/optional vars, defaults, the .env→process bridges (dev.sh, docker-compose), offline AI mode |
| [keys/](./keys/README.md) | Operators | The host key store: one standalone six-part file per credential kind (where / propagated / set / tested / used / install guide), index + shared flow + two-layer testing (shape suite + live probes) + per-host state |
| [../SETENV.md](../SETENV.md) | First-timers | The environment operating manual: create the file, choose each key (`env.sh init` / `edit`), propagate to the components, prove it works |
| [deployment.md](./deployment.md) | Operators | docker-compose topology, image builds, ports, reverse-proxy expectations, current Railway public deploy |
| [backup-restore.md](./backup-restore.md) | Operators | The backup story: managed (Supabase toggle) + portable dumps (`scripts/backup-db.sh`), restore runbook, scratch-stack walkthrough |
| [releasing.md](./releasing.md) | Maintainers | How a `v*` tag becomes a release: the `ci` gate, the test step, the four builds and checksums, the attestation, the no-overwrite publish, what `insideout version` prints per build, how to verify a download (and what `bootstrap.sh` checks for you), rollback, the supply-chain controls, and the decisions still open (version policy, rollback of a published release, installs without `gh`) |
| [cli-mcp-and-guidance.md](./cli-mcp-and-guidance.md) | Users + agents | The `insideout` CLI (`scripts/install.sh` → `~/.local/bin` + PATH), the MCP server registration snippet, and the `insideout.yaml` guidance flow that feeds GitHub evidence into a roadmap |

For what the product *is*, see the root [README.md](../../README.md).

> **Note:** [local-development.md](./local-development.md) supersedes the old
> `docs/INSTALL.md` (removed 2026-07-21; see git history), which predated the
> Go rewrite and described a toolchain setup that is no longer how this repo
> is developed.
