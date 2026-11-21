# Installation

People start at the [user guide](./user-guide.md): the prebuilt binaries,
`insideout setup`, and sign-in. That path needs no environment variables.
The maintained map of the wizard is [setup paths](./setup-paths.md).

This page is the full reference. `insideout setup` with no flags, on a
terminal, asks numbered questions (Enter accepts the hosted service).
Flags drive the same door when there is no terminal. A server on your
own machine is [below](#your-own-server).

Print the hosted API the binaries call:

```bash
./scripts/default-api.sh
```

## Prebuilt binaries

No checkout, no Go toolchain — the bootstrap script downloads the
right `insideout` + `insideout-mcp` for your OS and architecture
(macOS/Ubuntu, amd64/arm64), verifies the sha256 checksum and, when it
can, the build provenance, and installs them the same way `install.sh`
does:

```bash
curl -fsSL https://github.com/frankji-groundcontrol/insideout/releases/latest/download/bootstrap.sh | bash
# or from a checkout: scripts/bootstrap.sh [install-dir]
```

After the checksum, `bootstrap.sh` runs `gh attestation verify` on the
archive when the GitHub CLI is installed, signed in to github.com and
new enough to have that command. The check is pinned to this repository
and to the `release.yml` workflow that builds the release, and a failed
check installs nothing. Without a usable `gh` the script prints
`Provenance not checked` with the reason and installs on the checksum
alone, which shows the archive matches the `checksums.txt` beside it, not
who built it. `v0.1.0`, the only release so far, predates the
attestation step: with `gh` signed in the one-liner stops on it until a
tag cut by the current workflow exists, and `scripts/install.sh` from a
checkout (build from source) works meanwhile. An archive you built
yourself has no attestation either. How to verify by hand:
[releasing.md](./releasing.md#verifying-a-download).

`INSIDEOUT_RELEASE_BASE` overrides the download base for a self-hosted
mirror of a release, and must be an `https://` URL: plain http is accepted
only for a mirror on this machine (`localhost`, `127.0.0.1` or `[::1]`),
and a redirect to plain http is refused. Anything else stops the script
before it downloads. This path is live: `v0.1.0` (2026-10-11) publishes
all four targets and the anonymous one-liner above was verified working
the day the repository went public, before the provenance check existed.

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
[cli-mcp-and-guidance.md](./cli-mcp-and-guidance.md#mcp-server).

Signup on the hosted service is invitation-only. Register with an invitation
code, then sign in with a password, emailed code, or magic link:

```bash
export INVITE_CODE='invite-code'
export ACCOUNT_EMAIL='you@example.test'
insideout register --invite "$INVITE_CODE" "$ACCOUNT_EMAIL"
insideout login --code "$ACCOUNT_EMAIL"
insideout login --magic "$ACCOUNT_EMAIL"
```

An invitation code comes from the server's administrator. Until the owner
picks a permanent route (rv4-USER-12), a stranger asks the way the
[README](../../README.md) says: an "Invite request" issue without an e-mail
address, or the repository owner through GitHub. With no code you can still try
the product on a server of your own: from a checkout, `insideout setup --local
--postgres bundled` (it needs Docker; see [Your own server](#your-own-server)).

Confirm the hosted path without installing again:

```bash
./scripts/setup.sh --hosted
```

## Your own server

You need PostgreSQL 18 (the major CI tests) and, for the bundled database, Docker.
The two values the server refuses to boot without are `DATABASE_URL` and
`INSIDEOUT_JWT_SECRET` (at least 32 characters). `insideout setup --local`
writes both. Full variable reference: [environment.md](./environment.md).
Day-to-day commands: [local-development.md](./local-development.md).

### Guided setup

`insideout setup` on a terminal offers this path as choice 3. The flags
below are the same work without a terminal (`./scripts/setup.sh` before
the binary exists).

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
[local development](./local-development.md) and must never run against a
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

The hosted path is [the setup skill](../../skills/insideout-setup/SKILL.md).
A server on this machine is [the server skill](../../skills/insideout-server/SKILL.md).
Do not read `.env`. Do not print secret values, tokens, or database URLs.

1. If they did not ask to run a server, install the prebuilt binaries
   ([user guide](./user-guide.md)) and run `insideout setup` on a terminal,
   or `insideout setup --hosted` without one. Merge
   `{ "mcpServers": { "insideout": { "command": "insideout-mcp" } } }`
   into the agent MCP config. Do not set `INSIDEOUT_API`. Do not invent
   a token. Sign-in is
   `ACCOUNT_EMAIL='you@example.test'; insideout login "$ACCOUNT_EMAIL"` with
   the password on stdin.
2. Own server, only when they asked: bundled Docker Postgres, or a URL
   they already have.
   - Bundled: `insideout setup --local --postgres bundled`
   - Existing URL: `insideout setup --local --postgres own` with the
     URL on stdin.
   - A fresh own-URL database needs the two roles; add `--create-roles`
     when the DSN's role may create them (stdin carries the DSN, so
     setup cannot prompt), or run the SQL it prints.
3. After a local server exists, set
   `INSIDEOUT_API=http://127.0.0.1:8080/api/v1` or the CLI still calls
   the hosted API.
4. First admin: on a disposable local database, run `insideout seed`, then
   promote `demo@insideout.local`; on a shared or hosted server, run
   `ACCOUNT_EMAIL='you@example.test'; ./scripts/dev.sh -C server go run ./cmd/insideout promote-admin "$ACCOUNT_EMAIL"`,
   which creates the account (password read from stdin) when it does not exist
   and promotes it.
5. Verify everything with `insideout setup --check`.

Non-interactive `insideout setup` (or `setup.sh`, which forwards to it) with no flags exits 2 and names the flags.
