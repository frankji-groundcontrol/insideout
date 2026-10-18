# Installation

Two ways to get a working InsideOut. The CLI and the MCP server
already know the hosted API, so the usual path needs no environment
variables. A server on your own machine is a separate, guided setup.
Once the binary exists, `insideout setup` is the front door for both
paths — interactive on a terminal, flag-driven for agents.

Print the hosted API the binaries call:

```bash
./scripts/default-api.sh
```

## Prebuilt binaries

No checkout, no Go toolchain — the bootstrap script downloads the
right `insideout` + `insideout-mcp` for your OS and architecture
(macOS/Ubuntu, amd64/arm64), verifies the sha256 checksums, and
installs them the same way `install.sh` does:

```bash
curl -fsSL https://github.com/frankji-groundcontrol/insideout/releases/latest/download/bootstrap.sh | bash
# or from a checkout: scripts/bootstrap.sh [install-dir]
```

`INSIDEOUT_RELEASE_BASE` overrides the download base (self-hosted
mirrors, testing). This path is live: `v0.1.0` (2026-10-11) publishes
all four targets and the anonymous one-liner above is verified working
(the repository went public the same day).

## Hosted service

From a checkout of this repository (Go 1.26+, see `server/go.mod`):

```bash
./scripts/install.sh
insideout setup
```

`install.sh` builds `insideout` and `insideout-mcp` and installs
them into `~/.local/bin` (override with the first argument or
`INSIDEOUT_INSTALL_DIR`), appending one marked PATH line to your
shell rc when needed; it is safe to run again. `insideout setup`
then walks the remote path in a form: API base (hosted default) →
reachability probe → sign-in with password, emailed code, or magic
link (hidden as you type) → the token is saved to a 0600 config file
at `<OS config dir>/insideout/config.json` — no export lines to
copy. Process environment still wins where set (`INSIDEOUT_API`,
`INSIDEOUT_TOKEN`), so agents and CI are unaffected.

Agent form of the same door:

```bash
insideout setup --hosted    # confirm the hosted path without a form
insideout login you@example.com   # or the plain verb; password on stdin
insideout whoami
```

`insideout login` saves the same 0600 config file as the form (the
server it signed in to, the access token, and the refresh token that
renews it) and prints the file's path on stderr; stdout stays the bare
access token, so `INSIDEOUT_TOKEN="$(insideout login you@example.com)"`
still works. `insideout whoami` before any sign-in exits non-zero with
`unauthorized: not signed in; run insideout login`. That is a finished
install, not a missing server.

Register MCP. The entry needs no `env` or token: `insideout-mcp` reads
the server and token of the saved login, and the `auth_*` tools can
sign the process in:

```json
{ "mcpServers": { "insideout": { "command": "insideout-mcp" } } }
```

Cursor reads `~/.cursor/mcp.json` (global) or `.cursor/mcp.json`
(this project). Merge the `insideout` entry. Claude Code reads
`.mcp.json`, or run `claude mcp add insideout -- insideout-mcp`. A
client that does not inherit your shell `PATH` needs the absolute path
from `command -v insideout-mcp`. Reload MCP servers after writing the
file. Details:
[cli-mcp-and-guidance.md](cli-mcp-and-guidance.md#mcp-server).

Signup on the hosted service is invitation-only. Register with an invite
code, then sign in with a password, emailed code, or magic link:

```bash
export INVITE_CODE='invite-code'
export ACCOUNT_EMAIL='you@example.test'
insideout register --invite "$INVITE_CODE" "$ACCOUNT_EMAIL"
insideout login --code "$ACCOUNT_EMAIL"
insideout login --magic "$ACCOUNT_EMAIL"
```

Confirm the hosted path without installing again:

```bash
./scripts/setup.sh --hosted
```

## Your own server

You need PostgreSQL 18 (the major CI tests) and, for the bundled database, Docker. After the
binary/local server path is selected, use the complete
[key setup tutorial](key-setup-tutorial.md) and
[key testing tutorial](key-testing-tutorial.md) to configure and prove each
setting. The
two values the server refuses to boot without are `DATABASE_URL` and
`INSIDEOUT_JWT_SECRET` (at least 32 characters). Full variable
reference: [environment.md](environment.md). Day-to-day commands:
[local-development.md](local-development.md).

### Guided setup

`insideout setup --local` (or `./scripts/setup.sh` before the binary
exists — same flags) asks when stdin is a terminal and requires
flags otherwise, so an agent can drive it.

The setup command writes `.env` through the Go catalog writer
(comments preserved, 0600, values never displayed — the DSN is shown
redacted), generates the JWT secret, starts the bundled Postgres
when chosen, probes the database natively (pgx), applies migrations
in-process, offers to boot the API with the written values injected
into the server's environment, and prints a summary gate (what is
set, what rides a default, what is outstanding).

```bash
insideout setup --local --postgres bundled   # Docker Postgres on 127.0.0.1:5442
insideout setup --local --postgres own       # paste a DATABASE_URL (hidden)
insideout setup --local --postgres bundled --no-start   # write .env only
```

A **fresh** database needs the two NOSUPERUSER roles that
`docker/postgres-init/001-create-app-role.sh` creates for the bundled path:
`insideout_owner` must own the database and have BYPASSRLS, while
`insideout_app` has CONNECT and does not bypass RLS. The own-URL path asks
PostgreSQL before it migrates: with both roles
present it migrates; with both missing and a connecting role that can create
them it offers to (a prompt at a terminal, `--create-roles` otherwise),
generates the passwords into `.env` without printing them and rewrites the
DSNs; in every other case it stops before any migration and prints the SQL to
run ([issue](../issues/2026-10-07-own-url-fresh-db-bootstrap.md),
[changelog](../changelogs/2026-10-10-close-open-gaps/setup-roles-and-bundled-owner.md)).

The installed CLI keeps calling the hosted API until you opt into
this server:

```bash
export INSIDEOUT_API=http://127.0.0.1:8080/api/v1
./scripts/dev.sh -C server go run ./cmd/insideout
```

Registration is invitation-only on a fresh database. On a shared or hosted
server, run `promote-admin <email>` (server subcommand, owner DSN): when the
address has no account it creates one, reading the new password from stdin
(hidden and typed twice on a terminal, one line when piped), and promotes an
existing account without touching its password. Then mint invites. For a
disposable local development database, the documented seed account is the
shorter path:

```bash
./scripts/dev.sh -C server go run ./cmd/insideout seed
./scripts/dev.sh -C server go run ./cmd/insideout promote-admin demo@insideout.local
export INSIDEOUT_API=http://127.0.0.1:8080/api/v1
insideout setup                 # sign in to the seeded account; password is hidden
insideout signup-mode open      # optional, local disposable DB only
```

The seed command is strictly for a disposable local database; it creates the
public development demo account described in
[local development](local-development.md) and must never run against a
shared or production database. Restore `insideout signup-mode invitation_only`
after any local open-signup exercise.

`./scripts/env.sh init` is the same environment setup as a scrollable
prompt, if you would rather not use setup. `./scripts/env.sh check`
validates without printing values.

## Re-verify any time

```bash
insideout setup --check        # API, database (pgx), GitHub token, LLM key shape
insideout setup --check --llm  # legacy openai compatibility probe; current schemas need a coach turn
```

The API line asks as the user who is signed in (the saved login, which
renews itself like any verb, or `INSIDEOUT_TOKEN`): `signed in` passes, and
an expired or revoked login fails the check with the fix (`insideout login`).
Signed out, the probe is anonymous and a 401 is the pass.

Unreachable targets are reported as `skip` with a reason (offline-safe);
configured targets that answer wrongly fail the check.

### Database choices

Bundled Docker Postgres comes up with `docker compose up -d postgres`
on `127.0.0.1:${POSTGRES_PORT:-5442}`. The app role is `insideout_app`.
`setup.sh` writes the matching `DATABASE_URL`.

Your own PostgreSQL 18 can be a dedicated database or a shared
instance. Migrations stay inside the `insideout` schema and do not
write to other tenants' `public` objects. On a shared instance the
role should own only that schema. See
[database-and-rls.md](../architecture/database-and-rls.md).

## For an agent

Ask one question, then run the matching command. Do not read `.env`.
Do not print secret values, tokens, or database URLs.

1. Ask whether they want the hosted service or a server on this machine.
2. Hosted: `./scripts/install.sh`, then `insideout setup --hosted`
   (or the interactive `insideout setup` on a terminal). Merge
   `{ "mcpServers": { "insideout": { "command": "insideout-mcp" } } }`
   into the agent MCP config. Do not
   set `INSIDEOUT_API`. Do not invent a token. Sign-in is
   `ACCOUNT_EMAIL='you@example.test'; insideout login "$ACCOUNT_EMAIL"` with
   the password on stdin.
3. Own server: ask bundled Docker Postgres, or a URL they already have.
   - Bundled: `insideout setup --local --postgres bundled`
   - Existing URL: `insideout setup --local --postgres own` with the
     URL on stdin.
   - A fresh own-URL database needs the two roles; add `--create-roles`
     when the DSN's role may create them (stdin carries the DSN, so
     setup cannot prompt), or run the SQL it prints.
4. After a local server exists, set
   `INSIDEOUT_API=http://127.0.0.1:8080/api/v1` or the CLI still calls
   the hosted API.
5. First admin: on a disposable local database, run `insideout seed`, then
   promote `demo@insideout.local`; on a shared or hosted server, run
   `ACCOUNT_EMAIL='you@example.test'; ./scripts/dev.sh -C server go run ./cmd/insideout promote-admin "$ACCOUNT_EMAIL"`,
   which creates the account (password read from stdin) when it does not exist
   and promotes it.
6. Verify everything with `insideout setup --check`.

Non-interactive `setup.sh` with no flags exits 2 and points here.
