# 41 · Remediation execution contract

How the [remediation plan](40-remediation-plan.md) is executed by a fan-out of
agents. The plan says *what* to change; this file says *how* the work is split,
proven and merged. The goal prompt given to the coordinator is short on purpose
and points here.

Base: `main` @ `d53035a` (clean, pushed). The review read `9f0b1c7`; the later
commit `913d245` touched `server/cmd`, `server/internal/setup` and
`docs/usage`. Plan line numbers are therefore approximate: locate code by the
quoted text in the role files, not by line number.

Integration branch: `rv4/integration`, local only. Never push, never merge to
`main`, never deploy, never rotate a key, never rewrite history.

## Hard rules for every agent

- Never read or print `.env` or `.env.local`; redact secrets, hosts and project refs.
- Real PostgreSQL only, no mocks; write only inside the `insideout` schema.
- Files stay at or under 350 lines; split, or file a dated modularity issue.
- `docs/` prose is English only.
- Work only in worktrees under `/private/tmp/insideout-rv4/<cluster>`. Leave the
  main checkout and `/private/tmp/insideout-review4` untouched.
- Commit with explicit paths (never `git add .`), conventional messages, and the
  trailer `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`.
- Flutter: `pub get --enforce-lockfile`; never commit a rewritten `client/pubspec.lock`.
- Cite findings as `rv4-<ID>` (the ID namespace collides with reviews 1 to 3).
- Shared indexes (`docs/HANDOFF.md`, `docs/plans/README.md`,
  `docs/changelogs/README.md`, `docs/changelogs/index/2026-10.md`,
  `docs/issues/README.md`, `docs/learning/README.md`, `docs/review/README.md`)
  are edited by the integrator only. The one exception is `w3-docs-truth`, which
  owns the HANDOFF and plans claims of row 3.6; the integrator rebases onto it.
- Each cluster writes its own changelog fragment at
  `docs/changelogs/2026-10-18-rv4-remediation/<cluster>.md`.

## One workflow per wave

All four waves run back to back with no approval pause. A red wave gate, an
exhausted repair loop, or a row that needs an owner decision stops the run.

1. **Planner** (one agent). Read the wave's plan rows and write
   `/private/tmp/insideout-rv4/reports/clusters-w<N>.json`: per cluster the
   findings, concrete "Files owned" (found by grep on the quotes), `after`
   dependencies, gate commands, and a unique private-Postgres port from
   55440 to 55499. Prove file-disjointness; merge clusters that overlap and log
   the merge.
2. **Worker** (one per cluster, worktree isolation). Branch `rv4/<cluster>` off
   the wave base SHA. For each finding:
   - Write the failing test first and commit it alone: `test(rv4-<ID>): red ...`.
     Run it and paste the red output. If it cannot be made to fail, report
     `NOT_REAL` with the evidence and make no fix.
   - Fix in a separate commit `fix(rv4-<ID>): ...`; the test must go green.
   - A row marked `dir` in the plan gets a short design spike first (a note in
     the changelog fragment), then the implementation. If the direction needs an
     owner decision, report `DEFERRED_OWNER` and do not guess.
   - Edit only "Files owned". Write `worker-<cluster>.json` with, per ID, the
     status (`FIXED`, `NOT_REAL`, `DEFERRED_OWNER`, `PARTIAL`), the commits, and
     the red and green evidence.
3. **Private database.** Run `/tmp/rv4-pg-up.sh <port>` *without* `--migrate`
   (that flag is wired to the review worktree). From your own worktree run
   `go run ./cmd/insideout migrate` with the printed DSNs under `env -i`, then
   run the `DATABASE_URL`-gated tests. Tear down with `/tmp/rv4-pg-down.sh <port>`.
   Never use ports 55432 or 55480 or a DSN from the environment. Roles:
   `insideout_owner` (BYPASSRLS), `insideout_app` (NOBYPASSRLS). If the scripts
   are gone, recreate a private Postgres 18 cluster with the same two roles.
4. **Verifier** (a fresh adversarial agent per cluster, sees only the branch).
   - At the red commit the new tests fail; at HEAD they pass.
   - Mutation check: revert the fix hunk or flip the key condition, and the
     tests go red again.
   - The diff stays inside "Files owned"; no weakened existing test, no DB mock,
     no secret, every file at or under 350 lines, the changelog fragment present.
   - The cluster gates pass.
   - Write `/private/tmp/insideout-rv4/reports/verdict-<cluster>-r<N>.json` with
     `ACCEPTED`, `REPAIR` or `REJECTED` and the reasons.
5. **Repair loop.** Worker and verifier, at most three rounds. After that the
   cluster is `BLOCKED` and excluded from integration.
6. **Reports** are JSON on disk, so the run is resumable (`resumeFromRunId`).
7. **Integrator** (one agent, serial). In `/private/tmp/insideout-rv4/integration`
   on `rv4/integration`, `git cherry-pick -x` the `ACCEPTED` clusters' commits in
   dependency order; resolve conflicts minimally and re-run the affected tests.
   Run the full gates below on a fresh private database. A red gate stops the
   run. The next wave's base SHA is the integration head after its gates. Then
   record the per-ID dispositions in
   [the open-findings file](../../issues/2026-10-18-review-4-open-findings.md),
   update the plan's `.track.yaml` and history, and the shared indexes. Before
   wave 0, commit this contract on `rv4/integration`.

## Full gates (integrator, per wave)

```sh
(cd server && go build ./... && go vet ./... && go test ./... -race -count=1)   # DATABASE_URL set
(cd client && flutter analyze --no-fatal-infos && flutter test)
(cd client && TZ=Asia/Shanghai flutter test)
for f in scripts/tests/check_*.py; do python3 -I "$f" || echo FAIL "$f"; done
sh scripts/check-redaction.sh && sh scripts/check-docs-recorded.sh
```

Plus the wave's own "Check" paragraph in the plan.

## Clusters (about 25; the planner confirms file ownership)

**Wave 0 (1).** `w0-privacy`: rows 0.1 to 0.4. Scrub addresses and paths in
`docs/changelogs` and `hosts/*/host.yml`; add the address rule to
`scripts/check-redaction.sh`; remove the dead `.gitignore` exception; file the
0.4 history decision as an issue record. Not the dashboard, key rotation or a
history rewrite (owner).

**Wave 1 (9).**

| Cluster | Rows | Owns (first guess) |
|---|---|---|
| `w1-cli-front-door` | 1.1, 1.2, 1.5 | `store/invitations.go`, `cmd/insideout/main.go`, `cli.go`, `release.yml` |
| `w1-ci-gates` | 1.4, 1.6, 1.11, CI half of 1.15 | `ci.yml`, `ubuntu-install.yml` |
| `w1-clock-and-runes` | 1.3, 1.17 | `internal/api` view builders and validators, shared `tooLong` |
| `w1-auth-atomic` | 1.9, 1.10, cooldown half of 1.13 | `store/auth_tokens.go`, `api/admin.go`, `api/auth_cooldown.go`, invite dialog |
| `w1-apiclient-escape` | apiclient half of 1.13 | `internal/apiclient/*` |
| `w1-github-repo` | 1.14 | `internal/github/github.go` |
| `w1-client-polish` | 1.16, dialog half of 1.12 | `time_ago.dart`, `roadmap_node_menu.dart`, `project_dialogs.dart` |
| `w1-ops-docs` | 1.7, 1.8, bootstrap half of 1.12, 1.18 | `README.md:170`, `docker-compose.yml`, `scripts/bootstrap.sh`, `scripts/backup-db.sh` |
| `w1-misc-correctness` | 1.15 without CI | `go.mod`, `product_tools.go`, `agentcontext.go`, `scripts/env.sh`, `store/project_updates.go` |

**Wave 2 (4).**

| Cluster | Rows | Note |
|---|---|---|
| `w2-client-credentials` | 2.1 + 2.2 | one resolver and one token flow |
| `w2-proposal-apply` | 2.3 + 2.4 | idempotent apply plus the client contract |
| `w2-agent-scope` | 2.5 | **after** `w2-proposal-apply`; design spike first; enforced server-side |
| `w2-rls` | 2.6 | one migration, write-side RLS walk, refuse to serve on BYPASSRLS |

**Wave 3 (11).**

| Cluster | Rows | Note |
|---|---|---|
| `w3-coach-trust` | 3.1 | |
| `w3-stream-bounds` | 3.2 incl. COMP-04 | leave CLOSE-05 out (multi-replica) |
| `w3-auth-network` | 3.3 | |
| `w3-release-supply` | 3.4 | |
| `w3-privacy` | 3.5 | `PRIVACY.md` and the decision record only; retention and erasure stay `DEFERRED_OWNER` unless a decision is recorded |
| `w3-docs-truth` | 3.6 incl. COMP-08, plus the README hunks of 3.8 | owns the HANDOFF and plans claims |
| `w3-test-hardening` | 3.7 | |
| `w3-context-deadlines` | 3.9 | |
| `w3-first-run-copy` | 3.8 code | **after** `w3-context-deadlines` |
| `w3-github-evidence` | 3.10 incl. COMP-05 | |
| `w3-client-assets` | 3.11 | COMP-03 is unverified: run its first check before changing anything |

## After wave 3

One whole-change adversarial review of `git diff d53035a..rv4/integration` by
three independent lenses: correctness, security, and regression against the
`PRODUCT.md` promises. Repair what they confirm, run the full gates once more,
and take the integration head as final.

## Honesty rules

- CI-config rows (`ci.yml`, `release.yml`, `ubuntu-install.yml`, Actions pinning)
  can only be verified locally: YAML parse, `actionlint` if installed, and the
  described command run by hand. Mark them "pending first CI run after push".
- Do not report `FIXED` without a red-then-green pair. `NOT_REAL`,
  `DEFERRED_OWNER` and `BLOCKED` are valid, reportable outcomes.
- Out of scope, per the plan's "Not in scope yet": new product slices, the
  multi-replica server, the history rewrite, QUAL-07 splits and the Low batch.
  Do not run a fresh multirole review.

## Final output

The `rv4/integration` head SHA and commit count; a per-ID disposition table;
the lists of `NOT_REAL`, `DEFERRED_OWNER` and `BLOCKED` with reasons; the
full-gate output; the report paths; teardown confirmation for every private
Postgres port. No push, no merge. Wait for the owner.
