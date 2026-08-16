# 2026-07-28 — Hands-on .env key-set guide (docs/SETENV.md)

[environment.md](../usage/environment.md) is the variable *reference* — but
new arrivals still had to assemble the ordering, the two database setups,
and the proof-it-works steps themselves. New [`docs/SETENV.md`](../SETENV.md)
is the hands-on walkthrough: Step 0 create (`./scripts/env.sh init` or
`cp .env.example .env`) → Step 1 `DATABASE_URL` (remote vs bundled-postgres
decision, the `pgbouncer=true` pooler substring, the multi-tenant schema
caution) → Step 2 JWT secret (`openssl rand -hex 32`) → Step 3 AI provider
(empty token = offline coach, a choice not an error) → Step 4 local-dev
toggles → Step 5 compose-only vars → Step 6 frontend (needs nothing by
default) → Step 7 prove it (migrate, boot, `TestAuthz`, smoke; plus a
fail-fast error → fix table). It closes with a security checklist and two
minimal placeholder `.env` skeletons, one per database setup.

## What changed

- **Added [`docs/SETENV.md`](../SETENV.md)** — English-only per the docs
  convention; links out to the reference for variable semantics rather than
  duplicating tables. Documents the `dev.sh` → `env.sh check` preflight
  ([added the same day](2026-07-28-envsh.md)) and is explicit about which
  commands load `.env` through the wrapper vs on their own
  (`server/scripts/smoke.sh` self-locates it).
- **Cross-links wired:** [docs/index.md](../index.md) map entry and
  [usage/README.md](../usage/README.md) table row; the environment-reference
  lines in [CLAUDE.md](../../CLAUDE.md) and [AGENTS.md](../../AGENTS.md) now
  name it; `environment.md`'s quickstart and bridge description,
  `local-development.md`'s Environment section, the [HANDOFF.md](../HANDOFF.md)
  verify block, and the [`.env.example`](../../.env.example) header point at
  `env.sh init|check` and the guide.

## Verification

- Every relative link resolves to an existing file; prose cross-checked
  against the verified facts — the fail-fast strings in
  `server/internal/config/config.go`, the `:?` guards in
  `docker-compose.yml`, the offline-mode log line, and the bundled-postgres
  URL/password match.
- Built through an author → adversarial verify → fix workflow: role-split
  corrections (dev.sh = run, env.sh = set up/validate) and one
  command-accuracy fix applied before merge.

## Operator notes

None — docs only, no behavior change.
