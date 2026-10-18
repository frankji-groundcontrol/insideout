# 2026-10-18 — Multirole review 4 and its remediation plan

Docs-only change: no code moved.

## What changed

- [Review 4](../review/2026-10-18-multirole-4/README.md) read `main` @
  `9f0b1c7` from a clean detached worktree: 13 reviewers (5 roles, 8
  dimensions), 8 skeptic clusters on the 23 Highs, a coordinator
  reproduction pass on private PostgreSQL 18.4 clusters, and a completeness
  critic. 145 findings after dedup: 0 Critical, 10 High, 88 Medium, 47 Low.
  Nine Highs were skeptic-confirmed, 14 more were downgraded to Medium, 0
  refuted; the tenth High (COMP-02) came from the critic and was reproduced
  by the coordinator. Scores run 5.8 to 8.0 (mean 6.69 as reported, 6.24
  after the critic's reconciliation).
- The folder is new under `docs/review/2026-10-18-multirole-4/` (README,
  executive summary, register, thirteen role and dimension files, critic,
  [remediation plan](../review/2026-10-18-multirole-4/40-remediation-plan.md))
  and is indexed from [docs/review/README.md](../review/README.md).
- The plan is ordered by impact over effort: Wave 0 privacy and owner
  decisions, Wave 1 eighteen one-commit fixes, Wave 2 six structural fixes,
  Wave 3 eleven trust, reliability and docs-truth rows; every one of the 145
  IDs sits in a row or on the "not in scope yet" list, and each wave ends in a
  runnable check.
- [Issue record](../issues/2026-10-18-review-4-open-findings.md): the ten
  Highs, the red test on `main` and the owner decisions, each linked to a
  register ID and a plan row. No `BUG-NNN` number was consumed (the next is
  still BUG-031).
- [Learning note](../learning/2026-10-18-a-wall-clock-test-needs-a-scheduled-run.md):
  a wall-clock test turns red by date, and a push-only CI never sees it.
- [Plan record](../plans/2026-10-18-multirole-4.md) and its board row;
  HANDOFF carries the resume path.

## What the review found, in one paragraph

The hardened core held (no Critical, 14 of 17 review-3 Highs verified closed,
`FORCE` RLS on 16 tables, real-database suite green except one test). The
edge and the gates did not: a fresh install cannot create its first account,
`--help`/`version` fail with a server-config error, a saved CLI login dies
after 15 minutes, four resolvers disagree about the API base, web "Expand
with AI" never works, re-accepting a proposal duplicates nodes, MCP hands an
agent the human-only Commit and decision tools, `main` has been red since
2026-10-16 (a wall-clock test; CI has no schedule), `release.yml` publishes
without a test, and a UTC+ deadline displays a day early.

## How it was verified

- Every High went to skeptics told to refute by default; the coordinator
  gave them its run results. 9 CONFIRMED, 14 DOWNGRADED, 0 REFUTED.
- Reproductions R1 to R15 are in the
  [README table](../review/2026-10-18-multirole-4/README.md#bugs-reproduced-by-running-the-code):
  the red test on a clean checkout (25 packages ok, 855 pass lines, one
  failure), the missing first-admin path, the SQL-level self-insert as admin,
  the dead `--api` flag, 52 MCP tools, `--help` failing, the ungated release,
  the link scan, `flutter analyze` and 458 client tests, and two after the
  critic (the `TZ=Asia/Shanghai` deadline and the byte-length title limit).
- A key-shaped scan and an identifier scan ran over the folder and the
  records; the result is pasted in the README.
- No code changed, so no build or test gate applies to this change.

## What was not done

- No fix was applied. The repository is unchanged apart from documents.
- COMP-03 (fonts and CanvasKit from Google CDNs reaching mainland China) is
  unverified: the review made no network calls. Accessibility, native
  Android and iOS targets and PostgreSQL 14 were not exercised
  ([critic §"could not cover"](../review/2026-10-18-multirole-4/30-completeness-critic.md)).
- The critic recommends restoring `DATA-01`, `USER-02` and `AIDEV-01` to
  High. The skeptic verdicts stand in the register; the plan already puts the
  first two in rows 2.6 and 1.2.

## For the operator

Nothing is required to keep running. Owner decisions the review surfaced (plan
Wave 0): the hosted admin address in five docs (PUB-01), the legacy Supabase
publishable key (PUB-05), whether to rewrite history that carries
institutional author identities (PUB-06), whether `hosts/` ships as an example
tree (PUB-10), and the privacy notice and retention posture (PUB-02, DATA-06).
Do not tag another release before plan row 1.5, which makes `release.yml` run
the tests.
