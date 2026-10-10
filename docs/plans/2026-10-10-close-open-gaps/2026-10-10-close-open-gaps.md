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
- [ ] Backend: coach split, per-iteration sections, history `LIMIT` in SQL.
- [ ] Backend: anthropic split.
- [ ] Backend: store roadmap split; build-from-PRD pre-check.
- [ ] Backend: apiclient split.
- [ ] Backend: setup split; own-URL role probe and provisioning.
- [ ] Backend: revision lock, membership check, workspace counts, reaper index,
      join rate limit.
- [ ] Env catalog parity test.
- [ ] Client: `api/client.dart` and `api/models.dart` splits.
- [ ] Client: `project_page` and `roadmap_canvas` splits; canvas items.
- [ ] Client: popup menu theme, FAB, coach errors, slab surfaces, low items.
- [ ] Client: `theme/block_*` and l10n table splits.
- [ ] Web walk and symbolized console check on a local stack.
- [ ] Re-triage every issue record; move resolved ones; record owner-only items.
- [ ] Adversarial verification of the whole change set.
- [ ] Push, fast-forward MBP, verify three-way sync, CI green.

## Expected to need the owner

Re-checked at the end; an item leaves this list only with evidence.

- Production deploys (`railway up`) and the signed-in hosted page load.
- Purging the 582 legacy users (destructive, production data).
- Android release build: no Android SDK, and a signing keystore is a secret.
- Running the install skill on real Ubuntu hardware.
- Deleting the local `app/.env` (Wave 0.12), and the Supabase backup runbook.
- MBP's Flutter SDK, on a machine that is currently overloaded.

## Status

In flight 2026-10-10. Phase 1 is done: CI is green on `8ee97fc`, MBP is on
`main`'s upstream with HTTP/1.1, and the lockfile guard is in. The backend and
client waves are next.
