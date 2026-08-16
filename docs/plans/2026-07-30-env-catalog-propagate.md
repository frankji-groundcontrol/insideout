# 2026-07-30 — Env catalog, TUI, and contract-scoped propagation

Bring `scripts/env.sh` and `docs/SETENV.md` up to the design recorded in a
sibling repository's practice notes — `env-management-design.md` (the ten
rules) and `verifying-what-fails-closed.md` (how to test them), plus its
`env.sh` / `env-write.sh` / `env_catalog.py` / `env_tui.py`. Those files live
outside this repo, so they are named rather than linked; the rules that matter
are restated in [docs/SETENV.md](../SETENV.md) and
[the verification practice note](../practices/2026-07-30-verifying-a-tty-only-tool.md).
Adapted to this repo's two components: `app` (Nuxt) and `server` (Go).

Three deliverables: an `edit` TUI, a `propagate` verb, and a `.env.example`
that is honest enough to be read by a machine.

## Ground truth established before editing

| Question | Answer | Evidence |
| --- | --- | --- |
| Does the Go side read a `.env` file? | **No.** `os.Getenv` only, no dotenv dependency. | `server/go.mod` has no dotenv; `server/internal/config/config.go`, `internal/github/github.go` |
| Which vars does the Go side actually require? | `DATABASE_URL`, `INSIDEOUT_JWT_SECRET` (≥32) — nothing else blocks boot | `config.Load` fail-fast branches |
| Which vars does Nuxt read? | exactly one: `NUXT_API_INTERNAL_BASE` | `app/nuxt.config.ts:8` — the only `process.env.` read in `app/` |
| Does `app/.env` override the root `.env` exported by `dev.sh`? | **No — the opposite.** c12 3.3.4 `setupDotenv` assigns only when `targetEnvironment[key] === undefined`. | probe: exported `NUXT_API_INTERNAL_BASE` survived; an app-only key was applied |
| Is `server/.env` / `app/.env` gitignored? | Yes, both, checked **inside** each dir | `git -C app check-ignore -q .env` |

### The actual hazard here (not Piper's)

Piper's failure was later-wins: a stale per-app copy silently overrode a
corrected root value. Here the layering runs the other way, which produces a
**bistable** file instead:

- `./scripts/dev.sh -C app pnpm dev` — root `.env` is exported first, so any
  key it declares wins and the matching line in `app/.env` is inert.
- `cd app && pnpm dev` — nothing exports the root, so `app/.env` is the *only*
  source and wins alone.

`app/.env` today is hand-written and holds `NUXT_API_INTERNAL_BASE` pinned to
`http://127.0.0.1:8080/api/v1`. The root `.env` does not declare that key, so
the file is live under *both* invocations right now — and goes inert the moment
anyone sets it at the root. Whether a given line matters depends on a fact
nobody can see from the file. Same fix as Piper's: stamp the provenance and
verify it.

## Schema audit — `.env.example` contradictions found

The convention is machine-read, so a violation is a bug. Truth column is from
`config.Load` and `docker-compose.yml`'s `:?` / `:-` guards.

| Var | Truth | Was | Action |
| --- | --- | --- | --- |
| `DATABASE_URL` | required | uncommented | keep |
| `INSIDEOUT_JWT_SECRET` | required | uncommented | keep |
| `INSIDEOUT_ACCESS_TTL` / `_REFRESH_TTL` | optional (`15m` / `720h`) | **uncommented** | comment |
| `ANTHROPIC_BASE_URL` / `AI_MODEL` | optional (code + compose `:-` defaults) | **uncommented** | comment |
| `ANTHROPIC_AUTH_TOKEN` | optional — empty **is** the documented offline mode | **uncommented**, under a comment saying "leave empty" | comment |
| `SERVER_PORT` / `APP_PORT` / `POSTGRES_PORT` | optional (compose `:-`) | **uncommented** | comment |
| `POSTGRES_APP_PASSWORD` | required *only* for the bundled-postgres setup | **uncommented** | comment; `check` keeps the conditional guard |
| `POSTGRES_SUPERUSER_PASSWORD` | optional (compose `:-`) | **uncommented** | comment |
| `GITHUB_TOKEN` | optional | **absent** — but in `env.sh`'s `KNOWN_NAMES` | add commented |
| `NUXT_API_INTERNAL_BASE` | optional, read by `app` | **absent** — but in `KNOWN_NAMES` | add commented at root + as the `app` contract |

The last two are the "second list that drifts" anti-pattern: `env.sh`'s
hardcoded `KNOWN_NAMES` already knew two variables the skeleton did not
declare. Derive the list from the skeletons instead.

## Tasks

- [x] Read the four reference documents and five reference scripts
- [x] Establish ground truth (table above); back up `.env` and `app/.env`
- [x] `.env.example` — fix the required/optional form; no inline `#` after values
- [x] `app/.env.example` — recreate as the Nuxt component's contract
- [x] `scripts/env_catalog.py` — all catalog logic, `--list`-able
- [x] `scripts/env_tui.py` — thin curses renderer only
- [x] `scripts/test_env_catalog.py` + `test_env_writes.py` + `env_testlib.py`
      — assert battery (83), incl. linters over this repo's own skeletons
- [x] `scripts/env-lib.sh` — shared helpers (closes the line-budget issue)
- [x] `scripts/env-write.sh` — `init` + `propagate`
- [x] `scripts/env.sh` — dispatcher + `check` (now component-aware, stale-aware)
- [x] `scripts/dev.sh` — gate on `env.sh check <component>`
- [x] Verification: unit tests, guard battery against failing inputs, tmux TUI drive
- [x] `docs/SETENV.md` rewrite; `docs/usage/environment.md` alignment
- [x] Changelog, issue closure, index updates
- [x] Adversarial review (4 lenses x verify-by-reproduction): 30 claims,
      28 confirmed, all fixed — see the changelog's review section

## Decisions

- **No `server/.env`.** The Go binary reads process env only; generating a file
  nothing loads would be false confidence. `propagate server` says so.
- **No `*_FILE` indirection.** Piper needed it for multi-line PEMs. This repo
  has no multi-line secret, so the machinery is not ported.
- **Five states only** (`set` / `default` / `missing` / `placeholder` /
  `unset`). Value *validity* (the 32-char JWT floor, Go durations, DSN shape)
  stays in `check` — one authority, not two.
- **Hand-written copies are warned about, never clobbered.** `propagate`
  refuses a file it did not generate; that is the deliberate-override escape
  hatch.
- **Staleness fails only for the component being launched**, and warns
  otherwise, so a stale `app/.env` cannot block a `-C server` test run.
- **A placeholder in a required variable fails**, found during verification:
  both skeleton values pass `config.Load` (non-empty DSN, 33-character secret),
  so a verbatim `cp .env.example .env` exited 0 and `dev.sh` launched an
  environment that could only fail later with an opaque DNS error. In an
  optional variable a placeholder stays a warning.
