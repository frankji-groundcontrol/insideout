# 2026-10-10 — Close the open gaps

Burn down every open record in [`docs/issues/`](../../issues/README.md) and on
the [board](../README.md). Standing instruction from the owner: close the open
gaps and open issues until none are left.

## Outcome

Each open record is fixed and verified, or states the reason only the owner
can close it. CI on `main` is green. GitHub, MBP2, and MBP end on one commit.

## What was open (2026-10-10)

| Group | Items | How it is verified |
| --- | --- | --- |
| CI red on `main` | Go job ([BUG-021](../../issues/2026-10-09-bug-021-go-tests-share-one-config-file-on-linux.md)); a hidden skip-gate failure behind it; python key-store audit | The three CI jobs, plus a keyless-checkout run of the python job |
| Host drift | hooks off on MBP2; MBP has no upstream and a flaky HTTP/2 route; MBP Flutter 3.47.0 vs the 3.44.x pin | `git config`, five clean `ls-remote` runs, a lockfile check |
| Backend defects | build-from-PRD runs the LLM before its conflict guard; `CreateRevision` race; `ListMembers` check; coach history over-fetch; per-iteration prompt sections; reaper index; workspace counts; join rate limit | Real-PostgreSQL tests on a disposable cluster, then CI |
| Backend feature | own-URL setup does not probe or provision the two roles | A scratch cluster with and without the roles |
| Env catalog | Go and Python readers of one schema can drift | A parity test in CI |
| Modularity | 13 source files over the ~350-line budget | Pure moves: build, vet, full test suites, and a moved-lines check |
| Client UI | block-craft fidelity follow-ups; canvas fit, keyboard pan, reduced motion, delete confirm, failure feedback | `flutter analyze`, `flutter test`, `dart format`, then a browser walk |
| Verification debt | block-craft web walk; W3-N2 minified console errors | A local stack and a real browser |
| Stale records | resolved issues still listed as open; Nuxt-era records about deleted code | Re-check each against the Flutter client and the code |

## Method

- **Verify against real PostgreSQL.** No Docker here, so a disposable
  PostgreSQL 18 cluster runs on a private port under `/tmp`, with the same two
  roles as `docker/postgres-init/001-create-app-role.sh`. One migrated base
  database is cloned per worker so parallel suites cannot see each other. The
  shared Supabase instance is never touched.
- **Local baseline first.** `go test ./...` against that cluster: 465 PASS,
  0 FAIL, 1 SKIP (the terminal-only test, run separately under a real pty).
- **Splits are pure moves.** No renames, no behavior change; the existing
  suites pin behavior, and a moved-lines check compares the line multiset
  before and after.
- **Workers own disjoint files** and commit in their own worktrees; records
  (changelog, issue index, board, HANDOFF) are written once, centrally.
- **Review-register sweep.** Three whole-tree reviews filed findings that no
  issue, plan, or changelog names by ID: 35 + 72 in the two multirole reviews
  (an ID in no record outside the review folders), and 80 Medium/Low in the
  frontend review (named neither in its own remediation plan, which was
  executed as a plan, nor anywhere else). Read-only agents decide, per
  finding, what is true at `HEAD`: fixed (with the closing line), parked
  feature, bounded defect, owner step, or not a defect. Every "open" claim and
  a sample of the "fixed" claims are checked against the code before anything
  is recorded, and each bounded defect is fixed test-first like the rest of
  this plan.
- **No deploys and no destructive data operations** unless the owner says so.

## Checklist

Mirrors the tracker (`.track.yaml`); the tracker is the source of truth.

- [x] BUG-021: pin `XDG_CONFIG_HOME` beside `HOME`; give the script test an
      absent config file.
- [x] Python key-store audit skips on a checkout with no keys.
- [x] CI skip gate: the docker test runs everywhere (empty `PATH`); exactly one
      named skip is allowed (`TestReadSecretTerminal`, needs a terminal).
- [x] gofmt six drifted files; gate gofmt in CI.
- [x] Enable git hooks on MBP2.
- [x] Disposable PostgreSQL 18 harness and a green local baseline.
- [x] Push the CI fixes and get all three CI jobs green (`8ee97fc`).
- [x] MBP housekeeping: upstream for `main`, repo-local HTTP/1.1.
- [x] Flutter SDK drift: `pub get --enforce-lockfile` in CI and the Dockerfile
      (confirmed by the next CI run; MBP's own SDK stays with the owner).
- [x] Backend: coach split, per-iteration sections, history `LIMIT` in SQL.
- [x] Backend: anthropic split.
- [x] Backend: store roadmap split; build-from-PRD pre-check.
- [x] Backend: apiclient split.
- [x] Backend: setup split; own-URL role probe and provisioning.
- [x] Backend: revision lock, membership check, workspace counts, reaper index,
      join rate limit.
- [x] Env catalog parity test.
- [x] Client: `api/client.dart` and `api/models.dart` splits.
- [x] Client: `project_page` and `roadmap_canvas` splits; canvas items.
- [x] Client: popup menu theme, FAB, coach errors, slab surfaces, low items.
- [x] Client: `theme/block_*` and l10n table splits.
- [x] Web walk and symbolized console check on a local stack (two passes,
      both themes at both widths; BUG-022, 025, 026, 028, 029).
- [x] Re-triage every issue record; move resolved ones; record owner-only items.
- [x] Review-register sweep: triage the 187 findings no record traces (107
- [x] Review-register sweep: fix the bounded findings test-first; record
      the rest with a fix prompt or the owner step.
- [x] Review-register sweep: disposition ledger, indexed from
      [`docs/review/`](../../review/README.md) — 187 findings, 0 unresolved.
- [x] Adversarial verification of the whole change set — every batch
      had its own verifier; a whole-set review covered the cross-batch
      seams and tree invariants.
- [x] Push, fast-forward MBP, verify three-way sync, CI green
      (`a84feb8`, run 38047747110 all green; GitHub, MBP2 and MBP on one
      commit with identical trees).

## Needs the owner

Re-checked 2026-10-10; an item leaves this list only with evidence.

- Production deploys (`railway up`) and the signed-in hosted page load. They
  also activate the [BUG-027](../../issues/2026-10-10-bug-027-nginx-resolved-the-api-upstream-once.md)
  nginx fix (its record has the two post-deploy requests) and the two new
  migrations (`ai_runs` reaper index; invite-code rotation).
- Purging the 582 legacy users (destructive, production data).
- A store-signed Android bundle: the keystore is a secret.
- Mail: `INSIDEOUT_MAIL_API_KEY` and `INSIDEOUT_MAIL_FROM` in Railway, then a
  redeploy.
- The GitHub App install, a live critic exercise (needs an LLM key), and
  Supabase's automated-backup toggle.
- MBP: Flutter 3.44.x, and a check that the Wave 0.12 leftover `app/.env` is
  gone (absent on MBP2).

Left this list with evidence: the install skill on Ubuntu (the
`ubuntu-install.yml` workflow runs it on an Ubuntu runner; green on `09c06fe`),
an Android build (the `android` job builds a release APK in CI), and the
Wave 0.12 `app/.env` on MBP2.

*Superseded 2026-10-11: most of this list closed that day — the deploy is
live with the BUG-027 checks passed, the legacy purge is done (859 junk
accounts), the mail key is re-keyed with the full funnel live, the GitHub
App is confirmed installed, the live LLM exercise ran (a real hosted
coach turn), and MBP is closed (Flutter 3.44.8, `app/.env` gone) — see
the [owner-gap plan](../2026-10-11-owner-gap-closure.md) and the
[mail-truth changelog](../../changelogs/2026-10-11-owner-deploy-and-mail-truth.md).
Still the owner's: the store-signed Android bundle (keystore) and the
Supabase automated-backup toggle ([HANDOFF](../../HANDOFF.md) item 2).*

## Status

Done 2026-10-10, modulo the push/CI/MBP delivery step in flight. The
review-register sweep finished end to end: all 187 untraced findings are
dispositioned in the
[ledger](../../review/2026-10-10-disposition-ledger/README.md) (65 fixed in
this wave test-first, 15 not a defect re-proved, 15 parked, 3 owner steps,
the rest already fixed before it), 24 worker batches each passed an
independent adversarial verifier (red/green reproduced, gates rerun,
mutations checked), and the whole change set — 86 commits — was fast-forwarded
onto `main` (`82cf072`). Integration itself cost three conflict resolutions
(two independently-written duplicate test files deduplicated, the nginx CI
job carrying both new checks) and two gated fixes (the route-auth scan folds
const-concatenated patterns; the roadmap node menu split into a part file
under the new line-budget guard).

Final gates on the integrated tree, fresh PostgreSQL 18 with all four sweep
migrations applied to a clean database: `gofmt`/`go build`/`go vet` clean;
`go test ./... -count=1` 814 PASS, 0 FAIL, 1 SKIP (`TestReadSecretTerminal`);
`go test -race` 25 packages, no races; Flutter `pub get --enforce-lockfile`,
analyze 14 infos (baseline 16), format clean, 421 of 421 tests;
`check_nginx_reresolve.py`, `check_nginx_headers.py` and
`check_nginx_webhook_body.py` all pass on the local nginx 1.27.5.

The orchestration froze once mid-wave (the original session died 16:18 local
with four agent workflows in flight) and was resumed losslessly from the
on-disk state; the lesson is recorded in
[learning](../../learning/2026-10-10-resumable-orchestration.md).

Delivered: pushed as `a84feb8` (89 commits over `2be918e`), CI green on the
new tip (run 38047747110, 5 of 5 jobs including the Go `-race` lane the sweep
itself added), and the MBP fast-forward done — GitHub, MBP2 and MBP verified
on one commit with identical trees. The whole-change-set adversarial review
(cross-batch seams, tree invariants, five re-proven red/fix pairs) returned
clean with zero findings. The plan is closed; what remains is the owner's.
