# Verification — What Was Actually Checked

Everything below is recorded in [TODO.md](../../TODO.md) (phases P1–P7) with
the corresponding checkboxes; nothing here goes beyond what that checklist
claims.

## Backend (Go)

- `go build`, `go vet`, and the unit test suite green: password hashing,
  JWT, refresh tokens, invite-code format, coach stage transitions, system
  prompts, and Anthropic SSE stream parsing (fixtures are real captured SSE
  payloads, not synthetic).
- Integration test suite (`server/internal/store/authz_test.go`) run against
  a **real PostgreSQL database**, covering the full authorization checklist
  **including deny paths** (ex-member lockout, self-privilege-escalation,
  no self-review of PRDs), plus the PRD review lifecycle.
- All 13 migration files in `server/db/migrations/` applied against the real
  shared-instance target; grants verified both ways: `insideout_app` can
  create/DML in `insideout`, cannot create in `public`.
  (Note: TODO.md P1 says "12" — the count on disk is 13; the file list is
  authoritative.)

## End-to-end (real HTTP)

- register → login (httpOnly cookies) → create workspace → join by invite
  code, against the real Go server + real PostgreSQL.
- `seed` command exercised the idea-conversion → PRD → coaching-conversation
  creation path against the real database.

## AI (real API)

- A full, real idea → PRD coaching exchange over SSE against a live
  Anthropic-compatible endpoint (user-supplied `ANTHROPIC_AUTH_TOKEN`):
  streamed reply received, message history persisted and read back
  ([TODO.md](../../TODO.md) P4).

## Data cutover

- Post-migration data visibility verified under RLS by querying **as a real
  migrated user** before the `juanleme` schema was dropped.

## Frontend

- `typecheck`, all frontend tests, and `pnpm build` pass.
- Light and dark themes browser-verified with screenshots; no hydration
  warnings.

## Packaging

- `docker compose build` verified for **both** the `server` and `app`
  images independently (after fixing the `pnpm-workspace.yaml` build-context
  omission, [BUG-006](../../issues/2026-07-20-bug-006-pnpm-ignored-build-scripts.md)).

## What remains open

- The single unchecked item in [TODO.md](../../TODO.md) is the P7 "Real-API
  smoke test" checkbox. P4's own entry records that the real coaching
  exchange **was** completed with a user-supplied token, so the P7 checkbox
  appears stale rather than unfinished — but it is left unchecked there and
  is reported as-is here.
- Known limitations (not verified because not built): avatar upload is a
  local-preview placeholder; theme/locale live in localStorage (possible SSR
  first-paint flash); PRD section editors are plain textareas.
