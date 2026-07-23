# 2026-07-23 — Bring the Go backend under version control

The entire Go backend in [`server/`](../../server/) (75 `.go` files, 15 SQL
migrations, `go.mod`/`go.sum`, multi-stage `Dockerfile`) had been built and
verified across the 2026-07-20 → 2026-07-22 changelogs but was **never tracked
by git** — the repo only tracked `app/`, `docs/`, and dotfiles. This change
tracks it and prepares it for a public push.

## What changed

- **Staged all of `server/`** (94 files): `cmd/insideout` (server + `migrate` +
  `seed`), `internal/{agent,api,auth,config,export,github,httpx,store}`, and
  `db/migrations`.
- **Added a Go section to [`.gitignore`](../../.gitignore)**: compiled binaries
  (`*.exe`/`*.dll`/`*.so`/`*.dylib`, `server/insideout`), `*.test`, coverage
  `*.out`, `vendor/`, and Go workspace files (`go.work`, `go.work.sum`).
- **Recorded the pre-commit review** as
  [a dated issue](../issues/2026-07-23-backend-precommit-review-findings.md).

## Review & verification

- **Security sweep (inline)**: no `.env`, secrets files, private keys, binaries,
  or large artifacts anywhere under `server/`. Config is fully env-driven
  (`DATABASE_URL`, `INSIDEOUT_JWT_SECRET`, `ANTHROPIC_*` read from the
  environment; JWT secret required ≥32 chars); no hardcoded DSN/host; the only
  credentials in the tree are intentional dev-seed values (`demo@insideout.local`
  / `demo12345`, non-routable `.local`). Safe for a public repo.
- **Adversarial code review** (workflow `backend-precommit-review`: 5 package
  reviewers + per-finding adversarial verification): **no push-blockers** — no
  secret leaks, no trivially-exploitable authz/injection. 2 medium concurrency /
  data-integrity bugs and 6 low hardening items were confirmed and deferred to
  [the findings issue](../issues/2026-07-23-backend-precommit-review-findings.md).
- **Health**: `go build ./...`, `go vet ./...`, `go test ./...` all green
  (agent, auth, export, store packages pass).

## Operator notes

No runtime or schema change — this only starts tracking existing, already
verified code. The deferred findings are follow-ups, not regressions.
