# CLAUDE.md — InsideOut

Guidance for Claude Code in this repository. Keep this file a thin router:
short rules and links, with the substance in modular docs under `docs/` (see
the [doc map](docs/index.md)).

InsideOut is a Go + PostgreSQL + Nuxt 4 SSR app for tracking others'
projects, capturing ideas, and refining them into PRDs with an agent coach.

## Before editing

1. Read the [doc map](docs/index.md) and the relevant
   [architecture doc](docs/architecture/index.md) for the area you touch.
2. Read the concurrent task board in [docs/plans/README.md](docs/plans/README.md).
   For multi-step work, open a dated plan there **before editing**, drive the
   task from its checklist, and keep the board's status, priority, next action,
   and blocker current.
3. Read [docs/HANDOFF.md](docs/HANDOFF.md) only for the concise resume path and
   worktree warnings; use [docs/TODO.md](docs/TODO.md) for known limitations.
   Do not duplicate plan history or detailed task status in the handoff.

## Key commands

```bash
(cd server && go build ./... && go vet ./... && go test ./...)              # backend: build, vet, unit tests
./scripts/dev.sh -C server go test ./internal/store/... -run TestAuthz -v   # real-DB integration tests (exports .env)
./scripts/dev.sh -C server go run ./cmd/insideout migrate                   # apply SQL migrations
./scripts/dev.sh -C server go run ./cmd/insideout seed                      # dev demo data
./scripts/dev.sh -C app pnpm dev                                            # frontend (pnpm, not npm)
(cd app && pnpm test && npx nuxi typecheck && pnpm build)                   # frontend checks
docker compose build
```

Environment reference: [docs/SETENV.md](docs/SETENV.md) (hands-on key setup), [docs/usage/environment.md](docs/usage/environment.md) (variables), [docs/usage/local-development.md](docs/usage/local-development.md) (workflow). `scripts/env.sh init|edit|check|propagate` sets, inspects and validates `.env` (never printing values) and generates `app/.env` from it; `scripts/dev.sh` preflights `env.sh check <component>` before launching, so a stale generated copy blocks that launch.

## Hard rules

- **Real verification**: no mocks — DB-dependent work is not done until the
  `DATABASE_URL`-gated integration tests pass against real PostgreSQL, and
  streaming endpoints are exercised live. See
  [docs/practices/](docs/practices/README.md).
- **Never write outside the `insideout` schema** — `DATABASE_URL` may point
  at a shared multi-tenant instance
  ([why](docs/architecture/database-and-rls.md)).
- **Privacy**: never read or expose `.env`/`.env.local`. In docs and
  examples, redact secrets, tokens, provider/project identifiers, database
  hostnames, and personal emails to placeholders. Never commit raw private
  transcripts or personal PII — hold such files out untracked (do **not**
  gitignore them) and record them in a local untracked-file manifest.
- **English-only `docs/` prose** (the 2026-07-20 plan folder stays bilingual
  as a historical record; quote non-English sources only when quoting is
  required).
- **Modular files**: keep files under ~350 lines and single-responsibility;
  no chunky monolithic scripts. When a chunky or mixed-responsibility file
  can't be split in the current task, record a dated modularity issue in
  [docs/issues/](docs/issues/README.md) with the target structure and a
  bounded fix prompt.
- Karpathy-inspired coding baseline: state assumptions, prefer the simplest
  useful change, edit surgically, define verification before claiming
  completion.

## Recording changes

Record meaningful changes under [docs/changelogs/](docs/changelogs/README.md)
— a dated file for ordinary changes, a dated folder for large changes. The
guardrail is active (`config/git-hooks/`, activate per clone with
`git config core.hooksPath config/git-hooks`): every source commit warns if
no changelog entry is staged; a `[checkpoint]`-tagged commit **blocks**
unless the changelog entry, the active `docs/plans/` file, and
`docs/HANDOFF.md` are staged. Keep the record system whole: update
[docs/architecture/](docs/architecture/index.md) when structure changes,
[docs/usage/](docs/usage/README.md) when commands/install/operator flows
change, [docs/learning/](docs/learning/README.md) for reusable lessons,
[docs/practices/](docs/practices/README.md) for repeatable methods — and the
nearest README index for every added or moved record. Verify links point at
existing files.

- **Design QA**: when the user posts comments about how a page looks or the
  frontend, record them in [docs/design-qa/](docs/design-qa/README.md) —
  append to the relevant dated file there (creating one if needed) and keep
  its README index current.

## References

- Doc map: [docs/index.md](docs/index.md)
- Agent contract: [AGENTS.md](AGENTS.md)
- Decision history: [docs/plans/2026-07-20-go-rewrite/](docs/plans/2026-07-20-go-rewrite/README.md)
- Bug book: [docs/issues/](docs/issues/README.md)
