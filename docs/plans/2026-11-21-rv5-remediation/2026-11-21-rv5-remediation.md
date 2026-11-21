# 2026-11-21 — Review 5 remediation

Execute the six waves of the
[review 5 remediation plan](../../review/2026-11-21-focused-review-5/40-remediation-plan.md)
as one goal. The owner started it on 2026-11-21 and answered the open
decisions the same day (recorded in the
[open-findings record](../../issues/2026-11-21-review-5-open-findings.md)):
per-person trees, a default view not a wall, delegates re-delegate
descendants only, agents may not change an owner, no Build from an unapproved
PRD, no completion override by an agent.

## Landing

- Branch `rv5/remediation`, cut from `main` @ `c2900fd` (`origin/main`). The
  in-flight install and password-reset slice that sat uncommitted on the MBP
  checkout is finished in wave 0 (row 0.4) and committed on this branch.
- After each wave passes its gates the branch is pushed. No workflow
  triggers on a push to a branch other than `main`, so a push runs no CI.
  **The owner asked on 2026-11-21 that CI not be run for this work**: the
  draft pull request opened for wave 0 (#23) was closed and no pull request
  is opened; verification is local only. `main` is never pushed from this
  plan.
- Code clusters run as dynamic workflows: a worker in its own git worktree,
  failing test first, then an adversarial verifier, then the coordinator
  cherry-picks onto the branch and re-runs every gate.
- Database work runs against a private PostgreSQL 18 cluster on
  `127.0.0.1`, one database per worker, roles created the way `ci.yml`
  creates them ([practice](../../practices/2026-11-21-probe-a-real-database-without-touching-the-tree.md)).
  Nobody uses `scripts/dev.sh` or `.env` here: they may point at the hosted
  database.

## Gates (every wave)

1. `(cd server && go build ./... && go vet ./... && test -z "$(gofmt -l .)")`
2. `go test ./... -count=1` with `DATABASE_URL` and `DATABASE_OWNER_URL` on a
   freshly migrated private database, and once without them.
3. `(cd client && flutter analyze --no-fatal-infos && flutter test)` with the
   analyzer info count at or under the CI baseline.
4. Every `scripts/tests/check_*.py` exits 0; `scripts/check-redaction.sh`
   exits 0.
5. `(cd client && dart format --output=none --set-exit-if-changed lib test)`
   reports no change (CI's format step, run locally).

## Checklist

- [x] Wave 0 · green tip, slice finished, concept written (rows 0.1 to 0.7) —
      [record](../../changelogs/2026-11-21-rv5-remediation/wave-0.md)
- [x] Wave 1 · data loss and misplaced writes (rows 1.1 to 1.8) —
      [record](../../changelogs/2026-11-21-rv5-remediation/wave-1.md)
- [x] Wave 2 · the task tree (rows 2.1 to 2.9) —
      [record](../../changelogs/2026-11-21-rv5-remediation/wave-2.md)
- [ ] Wave 3 · surface parity (rows 3.1 to 3.7)
- [ ] Wave 4 · structure (rows 4.1 to 4.7)
- [ ] Wave 5 · documentation index (rows 5.1 to 5.6)

Per-row dispositions are recorded wave by wave in the
[changelog folder](../../changelogs/2026-11-21-rv5-remediation/index.md).

## Status

Started 2026-11-21. Tracker:
[2026-11-21-rv5-remediation.track.yaml](./2026-11-21-rv5-remediation.track.yaml).
