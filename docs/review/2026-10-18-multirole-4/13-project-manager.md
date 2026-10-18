# 13 · Project manager

**Lens:** stranger path clone to first success, HANDOFF / board / issues / changelog coherence, CI coverage versus record claims, release hygiene, license and Go floor agreement, record system volume and staleness, privacy of public records, wall-clock sibling sweep for L1  
**Score:** **6 / 10**  
**Bottom line:** Real engineering discipline (real-Postgres CI, honest product pitch, nine unimplemented promises enumerated), but the record system over-claims and several safety nets are not connected: four doc/contract guards run nowhere, CI has no schedule so the L1 red was invisible while HANDOFF says the head run is green, the binaries have no help or version, and the "zero broken doc links" claim is false (76 broken).

Tree: `main` @ `9f0b1c7`. Every High this lens filed went to an adversarial skeptic cluster; where the skeptic changed a severity or corrected a claim it is marked **[Correction]** below. The full register is [01-findings-register.md](01-findings-register.md).

## Skeptic verdicts on this lens's Highs

| ID | Filed | Verdict | Final | Reason |
|---|---|---|---|---|
| PM-01 | High | DOWNGRADED | Medium | **[Correction]** The wiring gap is real. `git grep` for the four names outside docs/ hits only the scripts' own `Usage:` lines. ci.yml invokes only the nginx scripts and the env-suites job (`test_env_catalog.py`, `test_env_writes.py`, the unittest discover over tests/keys, `check-redaction.sh`). No Go test shells out to them, and … |
| PM-02 | High | DOWNGRADED | Medium | **[Correction]** Both claims are accurate. ci.yml:3-6 triggers only on pull_request and push to main, and smoke.yml's nightly cron (smoke.yml:17-18) runs smoke.sh with no `go test`; the only go test invocations are ci.yml:84 and ci.yml:111. docs/HANDOFF.md:68-69 does read '(the head run is green)'. Reproduced the failure scenario: `go test … |

---

## Scope

Project-manager lens on InsideOut at `main` @ 9f0b1c7 (clean detached worktree). I read README, PRODUCT status section, CONTRIBUTING, SECURITY, LICENSE (MPL-2.0), `docs/HANDOFF.md`, `docs/plans/README.md` (48 board rows, 72 files under `docs/plans/`), `docs/issues/README.md`, `docs/review/README.md`, the changelog indexes, all four workflows, `config/git-hooks`, `scripts/tests/`, the `v0.1.0` tag and the 731-commit log. I ran four things: a local build of both binaries and the first commands a stranger would type, a Markdown relative-link scan, the four `scripts/tests/check_*.py` guards, and a repo-wide wiring check for those guards. I started no Postgres, so no cluster needs stopping. I touched no network.

One disclosure: an early `go build` I ran with `GOFLAGS=-mod=mod` rewrote `server/go.mod` in the worktree. I restored it with `git checkout -- server/go.mod`, and `git status` is clean. The rewrite is itself evidence for PM-10.

## Scores

| Aspect | Score | Why |
| --- | --- | --- |
| Stranger path, clone to first success | 5.5 | The prebuilt one-liner works. The hosted default is invite-only with no invite path, and the installed binary has no `--help` or `--version`. |
| Record coherence (HANDOFF / board / issues / code) | 5.0 | Three records disagree on what is next. Frozen gate numbers have drifted. A "zero broken links" claim is false. |
| CI coverage vs claims | 5.5 | The real-DB lane is excellent. There is no schedule, four guards run nowhere, and the analyzer ratchet is slack. |
| Release hygiene | 5.0 | The `v0.1.0` tag exists and is annotated. There is no version identity in the binaries, no release gate, and no documented process. |
| Honesty of the product pitch | 8.0 | README defers to PRODUCT.md, and nine unimplemented promises are enumerated as issues. |
| Record system vs product | 4.5 | 510 Markdown files under `docs/`, and 170 of 731 commits are `docs:`. A board of 48 rows has none in flight. |
| **Overall** | **6.0** | Real engineering discipline. The records over-claim and the safety nets are not all connected. |

## What is good

- **Honest pitch.** `README.md:3-5` states it describes the running build and points at PRODUCT.md for the target. `PRODUCT.md:416-445` lists the unimplemented promises. Nine are tracked as dated issue records in `docs/issues/README.md` ("Open product slices").
- **CI proves behavior, not just compilation.** `ci.yml` runs the server suite on a pinned Postgres 18.6 with two CI-parity roles. It uses `-race` and turns any unexpected skip into a red build. It also has an upgrade-replay lane, an image-build lane, nginx checks, Flutter analyze/test/web-build, an APK build and a redaction guard.
- **The release path is real.** `.github/workflows/release.yml` builds four targets, writes a checksums file and ships `bootstrap.sh`. The tag is annotated and an ancestor of `HEAD`. `scripts/bootstrap.sh` rejects unsupported OS and arch with a clear message.
- **Contribution doors exist and are checked.** `CONTRIBUTING.md` and `SECURITY.md` are short and honest (private advisory, scope, out-of-scope). `scripts/tests/check_contribution_surface.py` passes.
- **The commit-msg hook gates checkpoints** (`config/git-hooks/commit-msg`), and the project's rules are mostly mechanized rather than aspirational.

## Findings

**PM-01 (High): four docs/contract guards pass today but run nowhere.**
- Evidence (confirmed by running them): `scripts/tests/check_go_floor.py:4: Usage: python3 -I scripts/tests/check_go_floor.py`. The same pattern holds at line 4 of `check_docs_truth.py`, `check_product_promise_gaps.py` and `check_contribution_surface.py`.
- A repo-wide grep finds no reference to any of these four outside their own file and dated records. `ci.yml`'s python job (lines ~296-316) runs only the env batteries and the redaction guard.
- Failure scenario: the records cite "docs-truth 0" as a gate (`docs/plans/README.md:12`), but a drift in the Go floor, tool count, or promise register is never caught. This is the same class as the L1 time bomb: a red-first guard nobody executes.
- Fix: add one step to the existing python job running the four scripts. Diff size: 6 lines.

**PM-02 (High): CI cannot see time-based breakage, and HANDOFF asserts a "green head" it cannot know.**
- Evidence: `.github/workflows/ci.yml:3-4: pull_request: / push: branches: [main]` has no `schedule`. `docs/HANDOFF.md:68: **public**, so Actions are free and CI runs normally (the head run is`, continued as "green)".
- The last CI run was 2026-10-11. `TestProjectQuietFlagged` (L1) flipped red around 2026-10-16 with no workflow able to notice. A rerun of the head is red today.
- Failure scenario: a future wall-clock, certificate, or dependency rot lands silently. The handoff tells the next agent the tree is green when it is not.
- Fix: add a weekly `schedule:` trigger to `ci.yml`, and make HANDOFF cite a commit SHA and date for the last green run, not "the head". Diff size: 3 lines.
- Sibling sweep for L1: I grepped Go and Dart tests for hard-coded 2026 dates. Every other one passes an explicit `now` (`roadmaptime`, `agentcontext`, `presence`, `auth_cooldown`) or uses past-only fixture dates (`client/test/*`). I found no second live bomb. The only unpinned `time.Now()` consumers are `projects.go:84` (the L1 case) and `roadmap.go:70`, whose tests do not hard-code deadlines.

**PM-03 (Medium): the next engineering work is invisible from the board and HANDOFF.**
- Evidence: `docs/HANDOFF.md:13: **Everything engineering is closed; the owner's account actions below are`. `docs/issues/README.md` lists nine open product slices and a recommended order in `2026-10-11-product-promise-gaps.md`.
- `grep -i "slice\|promise" docs/HANDOFF.md` returns nothing. In `docs/plans/README.md` the only slice hits are completed rows.
- Failure scenario: a new maintainer or agent reads HANDOFF, concludes only owner console actions remain, and never finds the nine promised-but-absent features. A red test (L1) also contradicts "closed".
- Fix: add one Pending board row per slice, or a single "Product promise slices" row. Add a HANDOFF "What to do next" item 0 that links the promise-gaps order and names the L1 repair.

**PM-04 (Medium): the installed binary has no help and no version, and a bare `insideout` demands a database.**
- Evidence (confirmed by running the built binary with `env -i`): `insideout`, `-h`, `--help`, `help`, `--version` and `version` all print `{"level":"ERROR","msg":"config","error":"config: DATABASE_URL is required"}` and exit 1. Source: `server/cmd/insideout/main.go:42: cfg, err := config.Load()`, reached because `server/cmd/insideout/cli.go:66: default:` falls through to server mode.
- Failure scenario: a stranger runs the one-liner from README, types `insideout --help`, and is told to supply a database URL. There is also no way to report which release is installed. `release.yml` stamps no version, and `scripts/bootstrap.sh` always installs `latest`.
- Fix: handle `help`, `-h`, `--help`, `version` and `--version` in `runClientCommand`. Add `-ldflags "-X main.version=$GITHUB_REF_NAME"` to the release builds.

**PM-05 (Medium): the release workflow publishes any `v*` tag without a gate or a documented process.**
- Evidence: `.github/workflows/release.yml:5: tags: ['v*']`. Line 62: `--title "$GITHUB_REF_NAME" --generate-notes`. There is no `needs:`, no test step and no check that the tagged SHA passed CI.
- `grep -ril "release process\|semver"` over `docs/usage` and `docs/practices` finds no release doc. The `v0.1.0` tag is 13 commits behind `HEAD`, and `releases/latest/download/` will serve it indefinitely.
- Failure scenario: tagging a commit that is red (for example today's `HEAD`) ships binaries, and anonymous installers receive them.
- Fix: add a first step to `release.yml` that checks the tagged SHA has a successful `ci` run. Add `docs/usage/releasing.md` covering the tag rule, version policy and rollback via `gh release upload --clobber`.

**PM-06 (Medium): the "zero broken doc links" claim is false.**
- Evidence: `docs/plans/README.md:12: zero orphan docs / zero broken doc links** verified by the coordinator's own gate runs (graph \`orphans=0 md_broken=0\` on 528 files`. The commit subject of 9f0b1c7 repeats it.
- My scan of tracked Markdown relative links found 76 broken. They are in changelogs 24, design-system 18, plans 13, review 13, design-qa 7, learning 1. Example: `docs/changelogs/2026-07-23-coach-markdown-and-positioning.md:11` points at the deleted `../../app/src/utils/markdown.ts`.
- Failure scenario: the closure record states a gate result the tree contradicts, so the next audit trusts a number that is not true. The gate's definition of "broken" (apparently excluding historical records) is not stated.
- Fix: state the exclusion in the claim, or run a link scan with an explicit allowlist of dated records in CI. First check: diff the gate's link count against `git ls-files '*.md'`.

**PM-07 (Medium): frozen gate numbers disagree, and the analyzer ratchet is slack.**
- Evidence: `docs/plans/README.md:14: server 26 pkgs / 1 allowed skip, client 456, analyze 14`. `.github/workflows/ci.yml:240: INFO_BASELINE: "16"`.
- The coordinator's run (L3, L4): 25 ok plus 1 FAIL, 458 client tests, 0 analyzer issues. The prose says 14, CI allows 16, and reality is 0.
- Failure scenario: the comment above the baseline says "never raise it", yet CI would accept 16 new infos on a tree with none, so the ratchet protects nothing. Counts frozen in prose rot within days.
- Fix: set `INFO_BASELINE` to the measured count after re-running `flutter analyze`. Replace prose counts with "see the last green `ci` run".

**PM-08 (Medium): stranger path to a first success is blocked at the hosted door.**
- Evidence: `README.md:182` says registration is invitation-only and returns 403 without an invite. The binaries default to the maintainer's hosted instance (`README.md:55`). Nothing says how to ask for an invite.
- The prebuilt one-liner (the easiest path, `README.md:109`) is third in order, after "from source" (`README.md:50`). The README install section never says that Windows is unsupported, although `scripts/bootstrap.sh` exits on any non-macOS or non-Linux OS.
- Failure scenario: a stranger installs, runs `insideout login`, gets no account, and has no documented way forward except a Postgres/Docker self-host.
- Fix: put the prebuilt one-liner first. Add a "Try it without an account" line pointing at `insideout setup --local --postgres bundled`, and add a one-line contact route for invites.

**PM-09 (Medium): the record system is the largest artifact.**
- Evidence: `docs/plans/README.md` rows reach 1,768 characters in a "Status" cell. All 48 board rows are terminal and none is in flight. `docs` holds 510 Markdown files, which is close to the count of Go and Dart source files (596). 170 of 731 commits are `docs:`.
- `docs/HANDOFF.md` promises "never history" but carries a run-on history paragraph (lines 13-27). It also contradicts itself: `docs/HANDOFF.md:63: **Flutter SDK drift:** MBP runs 3.47.0 and the repo pins 3.44.x` against `HANDOFF.md:45: **MBP:** closed 2026-10-11 — Flutter 3.44.8`.
- Failure scenario: the cost of keeping 4 or 5 records consistent per change is what produced PM-03, PM-06 and PM-07.
- Fix: archive terminal board rows into a dated `docs/plans/archive.md`, so the board lists only open work and a one-line status. Drop the stale drift warning.

**PM-10 (Low): `go.mod` marks a direct dependency as indirect, and no check catches it.**
- Evidence: `server/go.mod:22: github.com/charmbracelet/huh v1.0.0 // indirect`, while `server/internal/setup/ui/ui.go:13: "github.com/charmbracelet/huh"` imports it. Running with `-mod=mod` promoted it to the direct block.
- No `go mod tidy -diff` step exists in `ci.yml`.
- Fix: run `go mod tidy` once and add a tidy-diff step to the server job. Diff size: 2 lines plus a CI step.

**PM-11 (Medium): the public repo carries personal addresses against its own privacy rule.**
- Evidence: `docs/HANDOFF.md:37-38` prints the owner's address stems and a bounced rehearsal address with ellipses. `docs/changelogs/2026-10-11-owner-deploy-and-mail-truth.md:129: The flow account is` is followed by a full address on a personal-domain mailbox. The release tag also embeds a tagger address.
- `CLAUDE.md` requires personal emails to be redacted. `scripts/check-redaction.sh` only checks the dogfood address and the production host.
- Failure scenario: the repo went public on 2026-10-11, the same day these records were written, so the addresses are now indexable.
- Fix: replace them with `<email>`, extend the guard's pattern to cover them, and decide separately on history (public-release lane, L8).

**PM-12 (Low): the Ubuntu install lane and the Go-floor guard can disagree.**
- Evidence: `.github/workflows/ubuntu-install.yml:63: grep -E 'go1\.(2[5-9]|[3-9][0-9])'` accepts Go 1.25, but `server/go.mod:3` requires 1.26.0.
- `check_go_floor.py` does not scan workflows, and it is wired nowhere (PM-01).
- Fix: change the pattern to `go1\.(2[6-9]|[3-9][0-9])` and add the workflow to the guard's file list. Diff size: 1 line.

## Lead verification

| Lead | Result | Evidence |
| --- | --- | --- |
| L1 | Confirmed, not re-filed | Searched for siblings. None found beyond the one `time.Now()` consumer (see PM-02). |
| L2 | Confirmed and extended | `ci.yml:3-4` has no schedule. Extended with PM-02, plus the unwired guards (PM-01). |
| L3, L4 | Confirmed, extended | Counts differ from the prose (25+1 vs "26 pkgs"; 458 vs 456; analyzer 0 vs "14" vs CI baseline "16"). See PM-07. |
| L9 | Not examined | Outside this lens. |
| L10 | Extended | The release gate is also missing (PM-05). |
| L11 | Confirmed | `ubuntu-install.yml:63`, PM-12. |
| L12 | Not examined | Outside this lens. |
| L13 | Confirmed (76 by my scan, 74 by the coordinator) | PM-06. The closure record says `md_broken=0`. |
| L14 | Confirmed | `docs/issues/README.md` says next is BUG-031. |
| L16 | Not examined | Outside this lens. |
| L7, L8, L17, L18 | Not run | Not in this lens, except that PM-11 touches L8. |

## Recommendations

1. Wire the four `scripts/tests/check_*.py` guards into the existing python CI job (PM-01).
2. Add a weekly `schedule:` to `ci.yml`, and replace "the head run is green" with a dated SHA (PM-02).
3. Handle `help` and `--version` in `runClientCommand` and stamp a version in the release builds (PM-04).
4. Gate `release.yml` on a green `ci` run for the tagged SHA and write `docs/usage/releasing.md` (PM-05).
5. Add the nine promise slices to the board and to HANDOFF "What to do next", and fix the HANDOFF Flutter contradiction (PM-03, PM-09).
6. Correct or narrow the "zero broken doc links" claim, and set the analyzer baseline to the measured value (PM-06, PM-07).
7. Put the prebuilt one-liner first in the README and say how to get an invite or a no-account try-out (PM-08).
8. Redact the personal addresses, run `go mod tidy`, and tighten the Go pattern in the Ubuntu lane (PM-10, PM-11, PM-12).

## What this role would do first

Before anything else, I would repair the single test that is red today, then spend one small commit on PM-01 and PM-02: wire the four orphaned guards into CI, add a schedule, and replace the "head is green" sentence with a dated SHA. That one commit makes the safety nets the records already cite actually run. The next commit would be the release gate and the `--version`/`--help` fix, so a tag cannot ship red and a stranger's first command works. Only after that would I touch the record volume, because most of the doc work is already written and only needs consolidating.

**The release gate in one sentence:** cut the next tag only from a commit whose scheduled `ci` run, including the wired docs guards, is green after the L1 fix, and whose binaries answer `--version` with that tag.
