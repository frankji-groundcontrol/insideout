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
- [ ] Adversarial verification of the whole change set.
- [ ] Push, fast-forward MBP, verify three-way sync, CI green.

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

## Status

In flight 2026-10-10. Every item above has landed on `main`'s local history
(a fix lands behind its red test commit). Final gate on the
tip: Go `go test ./... -count=1 -v` on a scratch PostgreSQL 18, 580 PASS,
0 FAIL, 1 SKIP (`TestReadSecretTerminal`, needs a terminal), with `gofmt`,
`go build` and `go vet` clean; Flutter `pub get --enforce-lockfile`, analyze
(0 errors, 0 warnings, 16 baseline infos), `dart format` (194 files, 0
changed) and 269 of 269 tests; `scripts/check_nginx_reresolve.py` 6 of 6 on
nginx 1.27.5. What is left is delivery: push, CI on the pushed commit, and the
MBP fast-forward.
