# 2026-07-28 — dev.sh moves to the repo root and grows `-C <dir>`

The 2026-07-27 hygiene pass added `server/scripts/dev.sh`, which exported the
root `.env` for server-side commands only. Per the "one `.env` at the root,
one wrapper that propagates it to whichever submodule needs it" direction, the
script is now [`scripts/dev.sh`](../../scripts/dev.sh) at the repo root and
takes `-C <dir>`: the same export logic, runnable for any consumer. No
behavior change to what gets exported.

## What changed

- **Added root [`scripts/dev.sh`](../../scripts/dev.sh)** (executable):
  exports `.env` (`set -a; source; set +a`), `cd`s into `-C <dir>`, and execs
  the command — e.g. `./scripts/dev.sh -C server go run ./cmd/insideout` and
  `./scripts/dev.sh -C app pnpm dev`. Still never prints values and fails
  loudly when `.env` is missing; `-C` is required, so usage is unambiguous.
- **Deleted `server/scripts/dev.sh`** (superseded; `server/scripts/` keeps
  the self-contained `smoke.sh`).
- **Doc commands repointed, root-relative:** [CLAUDE.md](../../CLAUDE.md) key
  commands, the [HANDOFF.md](../HANDOFF.md) verify block,
  [local-development.md](../usage/local-development.md), and
  [environment.md](../usage/environment.md); the [`.env.example`](../../.env.example)
  header now names the two bridges. This also fixes a latent doc bug: bare
  `go run ./cmd/insideout migrate|seed` (and the Running section's
  `go run ./cmd/insideout`) never exported `.env`, so following them literally
  died on `config: DATABASE_URL is required` — they now go through the
  wrapper. And the frontend gains a propagation path: a non-default
  `NUXT_API_INTERNAL_BASE` in `.env` now reaches `pnpm dev` via
  `./scripts/dev.sh -C app pnpm dev` (Nuxt does not auto-load the root `.env`).

## Verification

- `bash -n` clean; no-args / missing-`-C` prints usage and exits 2.
- `./scripts/dev.sh -C server go version` and `./scripts/dev.sh -C app
  node --version` both exec with the real `.env` exported, printing nothing
  sensitive.
- Repo-wide grep: no live doc references the deleted `server/scripts/dev.sh`
  path (historical changelog mentions left intact).

## Operator notes

None for `.env` — its format is unchanged. Update any personal shell aliases
from `server/scripts/dev.sh <cmd>` to `scripts/dev.sh -C server <cmd>`.
