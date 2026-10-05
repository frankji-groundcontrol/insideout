# Host key store — the six-part reference

Every credential in this repository has a tracked YAML manifest and a
git-ignored `.key` value file under
[`hosts/<host>/keys/<module>/`](../hosts/README.md). This doc is the
usage reference for that store, structured in six parts per the
convention the agents_home family doc established: **where** the `.key`
files are, **how they are propagated**, **how to set them**, **how to
test them**, **how they are used**, and **where the installation guide**
covers them. Nothing here prints a value; the tooling never does either.

Hosts: `franks-macbook-pro.local` (the primary workstation) and
`buddys-macbook-pro-2.local` (the secondary, being filled key-by-key).

## 1. Where the `.key` files are

`hosts/<host>/keys/<module>/.secrets/<name>.key` — one line, no `KEY=`
prefix, mode 0600, dir 0700, git-ignored. The tracked
`<module>/<setting>.yaml` manifests beside them name every key
(`<name>: {var, kind}`). One module folder per function — never one
folder per key.

| Kind | Env var | Value source | franks-mbp | buddys-mbp-2 |
| --- | --- | --- | --- | --- |
| `jwt-secret` | `INSIDEOUT_JWT_SECRET` | self-minted (`openssl rand -hex 32`), one per environment | filled | filled 2026-10-05 |
| `database-app-dsn` | `DATABASE_URL` | Supabase connection string, `insideout_app` role | filled | env set — store pending |
| `database-owner-dsn` | `DATABASE_OWNER_URL` | Supabase connection string, `insideout_owner` role | filled | env set — store pending |
| `llm-api-key` | `INSIDEOUT_LLM_API_KEY` | LLM provider (deploy-only values live in Railway) | filled | env set — store pending |
| `mail-api-key` | `INSIDEOUT_MAIL_API_KEY` | Resend dashboard (deploy value in Railway) | filled | pending |
| `gh-webhook-secret` | `INSIDEOUT_GH_WEBHOOK_SECRET` | self-minted; same string in the GitHub App webhook config | filled | pending |
| `gh-app-id` | `INSIDEOUT_GH_APP_ID` | GitHub App settings page | filled | pending |
| `gh-app-private-key` | `INSIDEOUT_GH_PRIVATE_KEY` | GitHub App private-key PEM download (multi-line) | filled | pending |
| `gh-client-secret` | `INSIDEOUT_GH_CLIENT_SECRET` | GitHub App client secret (no server reader yet) | filled | pending |
| `postgres-app-password` | `POSTGRES_APP_PASSWORD` | self-minted; must match the app DSN's password | filled | pending |
| `postgres-owner-password` | `POSTGRES_OWNER_PASSWORD` | self-minted; must match the owner DSN's password | empty by design | pending |
| `github-token` | `GITHUB_TOKEN` | optional operator PAT | empty by design | pending |

Config-with-defaults variables (listen address, TTLs, CORS, LLM
base/model/schema, mail sender, public URL) are not credentials — they
stay in the `.env.example` catalog
([environment.md](environment.md)).

## 2. How they are propagated

The consumer is the root `.env` — the only file any process reads.
`scripts/dev.sh` exports it before every command, `docker-compose.yml`
interpolates `POSTGRES_*` from it, and `scripts/env.sh check` preflights
it. The store is the out-of-git backup: `python3 scripts/keys.py fill`
copies `.env` values into **empty-only** `.key` files (0600, dir 0700,
never prints values); `audit` reports filled/empty plus env drift. With
more than one host, scope both:
`INSIDEOUT_KEYS_HOST=<host> python3 scripts/keys.py audit|fill`.
Deployed values travel separately through the hosting provider's
variable store. One value bypasses the `.env` bridge by design: the
GitHub App PEM is consumed via `INSIDEOUT_GH_PRIVATE_KEY_FILE` pointing
straight at the `.key` file.

## 3. How to set them

1. Get the value (column "Value source" above). Self-minted ones:
   `openssl rand -hex 32` for the JWT secret and the webhook secret.
2. Put it in the live consumer: `./scripts/env.sh edit` writes `.env`
   interactively and never prints values. (The GitHub App PEM: leave
   `INSIDEOUT_GH_PRIVATE_KEY` empty and set
   `INSIDEOUT_GH_PRIVATE_KEY_FILE` to the `.key` path instead.)
3. Back it into the store:
   `INSIDEOUT_KEYS_HOST=<host> python3 scripts/keys.py fill` (empty-only;
   existing values are kept, never overwritten).
4. Rebuilding a fresh `.env` from scratch still goes through
   `./scripts/env.sh init`, pasting from the `.key` files — never paste
   a value into a tracked file.

Rotation reminders: the JWT secret is per-environment (≥32 chars); the
two compose passwords must equal the passwords embedded in the two
DSNs; the mail key is queued for rotation if session transcripts ever
leave the machine that holds them.

## 4. How to test them

- Whole `.env`: `./scripts/env.sh check` — validates without printing
  (it cross-checks the compose app password against `DATABASE_URL`,
  masks every secret, fails on placeholders).
- Owner DSN: `./scripts/dev.sh -C server go run ./cmd/insideout
  migrate` must apply cleanly.
- LLM key: boot the server — the
  `INSIDEOUT_LLM_API_KEY not set — using offline template-reply coach`
  notice must **not** appear.
- Mail key: request a magic link (HTTP 422 means the recipient was
  rejected, not a bad key).
- GitHub webhook secret: the GitHub App's delivery history shows signed
  deliveries answered 200.
- Store↔env drift: the scoped `keys.py audit` above.

## 5. How they are used

The Go server loads all of them from process env at boot
(`server/internal/config`): the app DSN is the runtime RLS-subject
connection; the owner DSN runs migrations and `scripts/backup-db.sh`;
the JWT secret signs access/refresh cookies; the LLM key selects the
real PRD coach over the offline template replier; the Resend key
switches auth mail from log-only dev mode to sending; the GitHub trio
(webhook secret, App id, App private key) verifies and authenticates
the evidence loop. The compose passwords provision the bundled
postgres's two roles. `GITHUB_TOKEN` (optional) raises evidence sync
above the unauthenticated rate limit. `INSIDEOUT_GH_CLIENT_SECRET` has
no server reader yet — kept so the credential is not lost.

## 6. Installation guide

The environment manual is [SETENV.md](../SETENV.md) with the
`env.sh init|edit|check|propagate` tooling and the `.env.example`
catalog; the first-time path is
[local-development.md](local-development.md). This doc adds only the
credential-storage layer on top of it. For the scheme's file layout see
[`hosts/README.md`](../hosts/README.md) and each host's
`keys/README.md` state table.
