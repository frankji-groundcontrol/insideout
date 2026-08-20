# Setting up your environment

One rule: **exactly one file holds real values** — the gitignored root `.env`.
Everything else is a committed skeleton (`.env.example`), a generated copy, or
a code default. Nothing secret appears inline in a command, a doc, or a commit.

This is the operating manual — *what to type*. For the meaning, default and
consumer of every variable see [usage/environment.md](usage/environment.md);
for the day-to-day dev workflow see
[usage/local-development.md](usage/local-development.md).

## Layout

```
insideout/
├── .env                  # gitignored — the ONLY file you author values in
├── .env.example          # committed skeleton; ALSO the schema (see below)
├── scripts/
│   ├── env.sh            # owns the environment: init/edit/check/propagate/list/redact
│   ├── env-lib.sh        #   shared helpers (sourced)
│   ├── env-write.sh      #   the two verbs that write files (sourced)
│   ├── env_catalog.py    #   all catalog logic — testable without a terminal
│   ├── env_tui.py        #   the curses screen; a thin renderer over the catalog
│   └── dev.sh            # owns the processes; gates every launch on env.sh check
└── client/               # Flutter; no .env — API_BASE is a dart-define
```

There is no `server/.env` or `client/.env`, and that is deliberate: the Go
binary reads process environment through `os.Getenv`, and Flutter web bakes
`API_BASE` at build time. `./scripts/dev.sh -C server <cmd>` (or `-C client`)
exports the root `.env` into the process instead. `env.sh propagate` skips
both rather than inventing files.

## First-time setup

```bash
./scripts/env.sh init      # interactive: creates .env from the skeleton, prompts
                           # only for what needs a choice (DSN input is hidden,
                           # offers to generate the JWT secret), regenerates
                           # then validates
./scripts/env.sh check     # re-verify any time
./scripts/env.sh check server
```

`init` never overwrites a value you already filled in, so rerunning it is safe.
The manual equivalent, same result without prompts:

```bash
cp .env.example .env && chmod 600 .env   # then edit the two required values
openssl rand -hex 32                     # -> INSIDEOUT_JWT_SECRET
./scripts/env.sh check                   # names + status only; values never printed
```

Verify nothing sensitive is trackable:

```bash
git check-ignore .env
```

## The skeleton is the schema

`.env.example` does three jobs, so there is no second list to drift out of sync:

| It answers | By |
| --- | --- |
| Which variables exist, for whom? | one skeleton per component; the union is the catalog |
| Required or optional? | an **uncommented** assignment is required; a **commented** one is optional |
| Did you choose this value, or copy it? | compare the live value against the skeleton's |

Only two variables are uncommented, because only two make `config.Load` refuse
to boot: `DATABASE_URL` and `INSIDEOUT_JWT_SECRET`. Everything else has a
working code or compose default, so marking it required would make `check`
report a non-problem — and a checker that cries wolf gets ignored.

That convention is machine-read, so a violation is a bug rather than a typo.
Two are enforced by `scripts/test_env_catalog.py`: the required set must equal
exactly those two names, and every declared variable must have a real consumer
(that second one exists because a `NAME` followed by `=` written inside an
explanatory *comment* parses as a declaration like any other — which is how a
phantom variable briefly entered these very skeletons).

### `set` versus `default`

A value byte-identical to the skeleton's came from the skeleton, not from a
person. That is surfaced as its own state — but **never** as an error, because
`15m` and `claude-sonnet-4-20250514` *are* the intended values. The things only
you can supply ship as `change_me…` / `your-remote-host`, so they surface as
placeholders and can never masquerade as a default.

```
✓ set          you chose this
= default      identical to .env.example — it came from the skeleton
! missing      required, nothing set
~ placeholder  still change_me… / your-remote-host
· unset        optional, running on a code default
```

A placeholder in one of the two **required** variables is a `check` **failure**,
so `dev.sh` will not launch. That matters because both skeleton values pass
`config.Load` — the DSN is non-empty and the secret is 33 characters — so a
bare `cp .env.example .env` would otherwise sail through the gate and fail
later with an opaque DNS error. In an optional variable a placeholder is only a
warning; nothing breaks.

## Step 1 — `DATABASE_URL` (required)

The one real decision. Pick exactly one of the two setups.

**(a) Remote instance** (dev default) — any Postgres 14+ host. `DATABASE_URL`
is the runtime role `insideout_app`. Migrations use `DATABASE_OWNER_URL`
(`insideout_owner`, NOSUPERUSER — never a superuser for DEFINER objects):

```
DATABASE_URL=postgres://insideout_app:<password>@<db-host>:5432/insideout?sslmode=require
DATABASE_OWNER_URL=postgres://insideout_owner:<password>@<db-host>:5432/insideout?sslmode=require
```

**(b) Bundled compose postgres** (self-hosted default) — start it first with
`docker compose up -d postgres`. On a fresh data volume
[`docker/postgres-init/`](../docker/postgres-init) creates `insideout_owner`
and `insideout_app`. Host port `${POSTGRES_PORT:-5442}`. App password must
equal `POSTGRES_APP_PASSWORD`; owner password must equal
`POSTGRES_OWNER_PASSWORD`:

```
DATABASE_URL=postgres://insideout_app:<app password>@localhost:5442/insideout?sslmode=disable
DATABASE_OWNER_URL=postgres://insideout_owner:<owner password>@localhost:5442/insideout?sslmode=disable
```

**Transaction poolers:** if the URL contains the substring `pgbouncer=true`
(PgBouncer/Supavisor transaction mode, e.g. Supabase port 6543), keep it. The
server detects that substring and switches pgx to the simple query protocol
([`server/internal/store/pool.go`](../server/internal/store/pool.go)), because
server-side prepared statements do not survive transaction pooling.

**Multi-tenant caution:** `DATABASE_URL` may point at a shared instance —
migrations and the app stay inside the `insideout` schema and never touch other
tenants' objects ([why](architecture/database-and-rls.md)).

## Step 2 — `INSIDEOUT_JWT_SECRET` (required)

HMAC signing secret for access/refresh JWTs, **minimum 32 characters**;
under that the server refuses to start. `openssl rand -hex 32` yields 64 hex
characters, well clear of the bar — or let `./scripts/env.sh init` generate and
store it for you. Generate a fresh value per environment; never reuse a dev
secret in production.

## Step 3 — AI provider (a choice, not a gap)

An empty `INSIDEOUT_LLM_API_KEY` **selects** the offline coach. The server logs
`INSIDEOUT_LLM_API_KEY not set — using offline template-reply coach` and runs a
deterministic template streamer — the PRD coach works end to end (stages, tools,
SSE) with no network and no cost. That is why the LLM lines are commented
in the skeleton. For a real provider, set `INSIDEOUT_LLM_API_KEY` (plus
`INSIDEOUT_LLM_BASE_URL` if your provider is not the default). Put any `/v1`
on the base URL yourself. `INSIDEOUT_LLM_MODEL` defaults to
`claude-sonnet-4-20250514`; `INSIDEOUT_LLM_SCHEMA` is `messages` or `responses`.

## Step 4 — Local-dev toggles

- `INSIDEOUT_COOKIE_SECURE=0` — set this when browsing over plain http, or the
  browser drops the auth cookies. Only the exact value `0` turns it off.
- `INSIDEOUT_DEV_CORS=1` — usually unnecessary. Hosted Flutter is
  same-origin via nginx; local Flutter web needs
  `INSIDEOUT_CORS_ORIGINS=localhost`. Only the exact value `1` turns
  permissive CORS on.
- `INSIDEOUT_ADDR` and the two TTLs rarely need changing.

## Step 5 — Only if you run docker compose

Skip for bare `go run` / `dev.sh` use.

- `POSTGRES_APP_PASSWORD` — required by compose's `:?` guard *for setup (b)
  only*, which is why the skeleton leaves it commented: a flat "required" would
  cry wolf for everyone on a remote DSN. A conditional requirement belongs in
  the checker, and `check` enforces it exactly when the bundled setup is in use.
- `POSTGRES_SUPERUSER_PASSWORD` — bootstrap only; the app never connects with it.
- `SERVER_PORT` / `APP_PORT` / `POSTGRES_PORT` — host-side mappings. Inside the
  containers the ports are fixed at `8080` / `3000` / `5432`.

## Seeing and setting everything

```bash
./scripts/env.sh edit          # curses list of every variable and its state
./scripts/env.sh edit --list   # the same rows, no terminal needed
```

↑/↓ move, Enter sets, `c` clears (writing the key back to its commented form),
`/` shows only what is outstanding, `q` quits. Each row carries one of the five
states above plus which component reads it.

Secrets are masked in **both** directions — shown as `••••••`, typed as
`******` — decided by the variable's *name* before any value reaches the
screen, because the decision has to be made before a widget could hold it.

It writes only the root `.env`, then offers to re-run `propagate` so the
components see the change. All of its logic lives in
[`scripts/env_catalog.py`](../scripts/env_catalog.py); the curses file only
draws. That split is what makes the behaviour testable at all — see
[verifying a tty-only tool](practices/2026-07-30-verifying-a-tty-only-tool.md).

## Propagating to the components

```bash
./scripts/env.sh propagate       # skips server and client (no dotenv files)
```

`propagate` still exists so a future component that owns a `.env.example`
can be generated from the root. Today it prints `skip` for `server` and
`client` and writes nothing. The Go server and Flutter client both take
configuration from the process environment / dart-define, not a nested
dotenv.

## Prove it

All commands from the repo root. The `dev.sh` ones preflight
`./scripts/env.sh check <component>` and export `.env` first.

```bash
./scripts/env.sh check --db                                # + a pg_isready probe
python3 scripts/test_env_catalog.py                         # catalog: read path
python3 scripts/test_env_writes.py                          # catalog: write path
./scripts/dev.sh -C server go run ./cmd/insideout migrate   # apply migrations
./scripts/dev.sh -C server go run ./cmd/insideout           # boot: listens on :8080
./scripts/dev.sh -C server go test ./internal/store/... -run TestAuthz -v   # real-DB RLS proof
./server/scripts/smoke.sh                                   # five-surface live smoke
```

`check` proves a variable is *present*; only a real request proves it is
*usable*. Both are needed and neither substitutes — `smoke.sh` is the end-to-end
half.

Fail-fast errors from `config.Load` (the server refuses to start):

| Error | Fix |
|-------|-----|
| `config: DATABASE_URL is required` | Set `DATABASE_URL` (Step 1) and export it via `dev.sh`. |
| `config: INSIDEOUT_JWT_SECRET is required` | Set a non-empty secret (Step 2). |
| `config: INSIDEOUT_JWT_SECRET must be at least 32 characters` | Use a longer secret. |
| `config: invalid INSIDEOUT_ACCESS_TTL "xyz": …` | Use a valid Go duration (`15m`, `720h`) or unset it. Same shape for `INSIDEOUT_REFRESH_TTL`. |

See [usage/environment.md § Troubleshooting](usage/environment.md#troubleshooting)
for the full list.

## Never print a secret

Every read-only mode is designed so its output is safe to paste into an issue:

- `list` — key **names** only
- `redact` — `KEY=<redacted>`
- `check` — a status per key: a secret as `set (N chars)`, a DSN with its
  userinfo blanked
- `edit` — masked by name, and `--list` renders the same masking

The rule is precisely *never print a **secret's** value*, and what counts as a
secret is decided by the variable's **name** (`KEY|SECRET|PASSWORD|TOKEN|…`,
plus `DATABASE_URL` and `INSIDEOUT_LLM_BASE_URL`, which embed or identify a
credential) — the decision has to be made before the value could reach a
widget, so it can never depend on inspecting the value. A name test therefore
always beats a content test: a real password that happens to contain the string
`change_me` is still masked.

Non-secret flags *are* echoed, deliberately: `warn: INSIDEOUT_DEV_CORS — 'yes'
— only exactly '1' enables it` is only actionable because it quotes what you
wrote. When a secret must be shown for orientation, a derived fact stands in
instead: its length, or the DSN with its password replaced. A DSN shape that
cannot be safely reduced — a libpq `host=… password=…` keyword string — is
masked whole rather than guessed at, and `check --db` strips the userinfo
before handing the DSN to `pg_isready`, because argv is not private.

## Hard don'ts

- **Never commit** `.env` (gitignored — check `git status --short` before
  every commit). Only the `.env.example` skeleton is tracked.
- **Never print values** in docs, chat, or issues. Use `env.sh redact`, and mask
  DSNs as `postgres://insideout_app:…@<db-host>:5432/insideout`.
- **Never put an inline `# …` comment after a value** in a skeleton — every
  consumer strips it, so the example value stops matching a live one and the
  set-vs-default signal breaks for that variable.
- Generate a fresh `INSIDEOUT_JWT_SECRET` per environment.
- On shared instances, stay inside the `insideout` schema.
- Keep `POSTGRES_SUPERUSER_PASSWORD` away from the app — it connects only as
  `insideout_app`.

## Related

- [usage/environment.md](usage/environment.md) — the variable reference:
  meaning, default and consumer of every variable.
- [usage/local-development.md](usage/local-development.md) — prereqs, running,
  testing.
- [Verifying a tty-only tool](practices/2026-07-30-verifying-a-tty-only-tool.md)
  — how these guards were tested, and why green unit tests were not enough.
- [Database conventions](architecture/database-and-rls.md) — `insideout_app`,
  RLS, the shared-instance rule.
