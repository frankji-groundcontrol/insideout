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

## Hosted service

From a checkout of this repository (Go 1.25+, see `server/go.mod`):

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

`insideout whoami` with no token returns `authentication required`.
That is a finished install, not a missing server.

Register MCP. The snippet omits the token until you have one; the
`auth_*` tools can sign the process in afterwards:

```bash
./scripts/mcp-config.sh          # hints + JSON
./scripts/mcp-config.sh -q       # JSON only
```

Cursor reads `~/.cursor/mcp.json` (global) or `.cursor/mcp.json`
(this project). Merge the `insideout` entry. Claude Code reads
`.mcp.json`. Reload MCP servers after writing the file.

Signup on the hosted service is invitation-only. Use an invite code,
or an emailed code or magic link (`insideout login --code`,
`insideout login --magic`).

Confirm the hosted path without installing again:

```bash
./scripts/setup.sh --hosted
```

## Your own server

You need PostgreSQL 14+ and, for the bundled database, Docker. The
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

One precondition the own-URL path does not handle yet: a **fresh**
database needs the two NOSUPERUSER roles provisioned first
(`insideout_owner` with BYPASSRLS owning the database, `insideout_app`
with CONNECT) exactly as `docker/postgres-init/001-create-app-role.sh`
does for the bundled path
([issue](../issues/2026-10-07-own-url-fresh-db-bootstrap.md)).

The installed CLI keeps calling the hosted API until you opt into
this server:

```bash
export INSIDEOUT_API=http://127.0.0.1:8080/api/v1
./scripts/dev.sh -C server go run ./cmd/insideout
```

Registration is invitation-only on a fresh database. Create the first
admin, then mint invites. For a local development database only, you
can open signup:

```bash
./scripts/dev.sh -C server go run ./cmd/insideout promote-admin you@example.com
./scripts/dev.sh -C server go run ./cmd/insideout signup-mode open
```

`./scripts/env.sh init` is the same environment setup as a scrollable
prompt, if you would rather not use setup. `./scripts/env.sh check`
validates without printing values.

## Re-verify any time

```bash
insideout setup --check        # API (401 = pass), database (pgx), GitHub token, LLM key shape
insideout setup --check --llm  # additionally one live provider request (GET /models)
```

Unreachable targets are reported as `skip` with a reason (offline-safe);
configured targets that answer wrongly fail the check.

### Database choices

Bundled Docker Postgres comes up with `docker compose up -d postgres`
on `127.0.0.1:${POSTGRES_PORT:-5442}`. The app role is `insideout_app`.
`setup.sh` writes the matching `DATABASE_URL`.

Your own Postgres 14+ can be a dedicated database or a shared
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
   `./scripts/mcp-config.sh -q` into the agent MCP config. Do not
   set `INSIDEOUT_API`. Do not invent a token. Sign-in is
   `insideout login <email>` with the password on stdin.
3. Own server: ask bundled Docker Postgres, or a URL they already have.
   - Bundled: `insideout setup --local --postgres bundled`
   - Existing URL: `insideout setup --local --postgres own` with the
     URL on stdin.
   - A fresh own-URL database needs the two roles provisioned first
     (see the note above).
4. After a local server exists, set
   `INSIDEOUT_API=http://127.0.0.1:8080/api/v1` or the CLI still calls
   the hosted API.
5. First admin:
   `./scripts/dev.sh -C server go run ./cmd/insideout promote-admin <email>`.
6. Verify everything with `insideout setup --check`.

Non-interactive `setup.sh` with no flags exits 2 and points here.
