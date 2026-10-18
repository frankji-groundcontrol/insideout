# 30 · Completeness critic

**Lens:** coverage, contradiction, score consistency, verdict-vs-measurement  
**Score:** **7.0 / 10** (how complete this review is)  
**Bottom line:** Review 4 is strong on server auth, RLS and closure accounting but never tested the zh-first runtime or native targets, and it under-scored write-side RLS and the broken front door; eight new findings are filed and the reconciled mean score is 6.24, not 6.69.

Tree: `main` @ `9f0b1c7`. The critic read every reviewer bottom line, finding list, skeptic verdict and the coordinator's reproduction results. It recommends severities but does not set them; its 8 missed finding(s) are in the [register](01-findings-register.md) as `COMP-NN`. Corrections the coordinator makes to the critic or to a reviewer are marked **[Correction]**.

---

## Completeness critic

Review 4 is thorough on server auth, RLS, closure accounting and the release path. It is thin on what a Chinese-language user and a native-app user actually meet, and on cost and abuse of the LLM path. Eight new findings (COMP-01 to COMP-08) follow; none repeats a digest row.

## Dimension not reviewed

Nobody tested the **zh-first runtime** that PRODUCT.md and the README advertise as first-class (FEAT-05 covers only export and convert locale). Spot-checking it found three defects:

- **Byte-length limits on text fields.** `server/internal/api/ideas.go:92` rejects `len(req.Title) > 200`. `len` counts bytes, so a Chinese title over about 66 characters is refused with "max 200 characters". The same check sits at 15 more sites (projects, workspaces, prds, roadmap, agent_routes). `project_updates.go:49` already uses `utf8.RuneCountInString`, so the fix pattern exists in-tree. (COMP-01)
- **Deadline date shifts a day for UTC+ users.** `roadmap_node_menu.dart:203` stores `picked.toUtc().toIso8601String()`, and `time_ago.dart:3` shows `iso.substring(0, 10)`. I measured with `TZ=Asia/Shanghai`: a picked 2026-10-18 renders as 2026-10-17. The roadmap is time-first, so the displayed deadline is the headline datum. (COMP-02)
- **Font and CanvasKit CDN dependence.** `client/web/index.html:25` loads Google Fonts and `client/nginx.conf:35` allows scripts only from the gstatic CanvasKit path. PUB-02 flagged this as privacy; nobody asked whether it loads for mainland users. I could not test the network, so this is a direction, not a repro. (COMP-03)

Also uncovered: **input-size and cost amplification** (COMP-04), **GitHub binding provenance for inbound webhooks** (COMP-05), **native Android/iOS targets** (COMP-06), and **backup file hygiene** (COMP-07). Accessibility got a score (fullstack 8.0 on a dimension) with no run of a screen reader or focus traversal, so treat it as unreviewed. Presence cursor POSTs are throttled only in the client (`roadmap_page.dart:67`, 200 ms); the server has no limit, which belongs with REL-01.

## README-vs-code contradictions missed

| Claim | Code | Filed? |
|---|---|---|
| PRODUCT.md:440 "every unimplemented promise is tracked as a dated issue" | Promise register rows 50, 100, 103, 116, 126 are PARTIAL with no issue; the tracker lists 9 slices | No (COMP-08) |
| PRODUCT.md progress is verifiable evidence | `roadmap.go:218` lets `in_progress -> done` with no evidence or reason; `SetProjectRepo` stores any URL unverified | No (COMP-08, COMP-05) |
| README.md:147 "any PostgreSQL 14+" (also local-development.md:12) | CI tests only postgres 18.6 (`ci.yml`) | No. Either add a 14 lane or say "18" |
| zh is first-class | byte caps, timezone shift, CDN-only assets above | No (COMP-01..03) |
| Env reference lists every variable | `INSIDEOUT_ENV_FILE`, `INSIDEOUT_ROLELESS_URL`, `INSIDEOUT_SETUP_REPO_ROOT` are read by code, absent from environment.md and .env.example | No (low; fold into docs-truth cluster) |

The `<html lang="en">` in export HTML and the missing `lang` in `index.html` are part of FEAT-05's family.

## Score consistency table

| Role | Reported | Reconciled | Reason |
|---|---|---|---|
| architect | 7 | 6.5 | Four resolvers and a stored token sent to the wrong base (R4) is a trust defect, not polish |
| ai-developer | 6 | 6 | Consistent with a confirmed High and an advisory-only policy |
| fullstack | 6.5 | 6 | Always-broken Expand button (R9); accessibility 8.0 had no evidence |
| project-manager | 6 | 6 | Consistent |
| user | 5.8 | 5.8 | Consistent; the front door is broken (R2, R6) |
| feature-implementation | 6.9 | 6.3 | Two confirmed Highs (human-only boundary, duplicate nodes) on the headline loop |
| code-quality | 8 | 7 | "No High" yet a 16-site byte-length bug, dead code, 36 unescaped URLs and a divergent IP helper |
| security | 6.8 | 6.5 | Confirmed login-code bypass plus unscoped repo and token findings |
| testing | 6.6 | 6.2 | Main is red (R1), release ungated (R8), no seam tests |
| reliability-ops | 7 | 6.5 | One user can exceed the 512m limit; health hangs; "no High" is generous |
| closure-audit | 7.4 | 6.8 | 3 of 17 closures partial, 11 docs-only "Fixed", 4 unwired guards |
| public-release | 6 | 5.5 | Real emails in a now-public repo (R11), unsigned ungated releases |
| data-rls | 7 | 6 | Write-side hole reproduced (R3); the headline strength is the read side only |

Mean reported 6.69; mean reconciled 6.24.

## Verdict-vs-measurement table

| ID | Skeptic | Recommend | Why |
|---|---|---|---|
| DATA-01 | High to Medium | High | R3 measured a non-owner self-inserting as admin and updating the workspace. "No API path" is a Go-side argument; the DB is the stated backstop. DATA-02 shows no test covers it |
| USER-02 (with FULL-03, PM-04) | High to Medium | High as one cluster | R6: help, version and typos all exit 1 with a server-config error. Three reviewers filed one root cause; counting it once at Medium hides a front-door defect |
| AIDEV-01 | High to Medium | High while AIDEV-04 stands | R5 shows an agent can accept its own proposal and commit, so an injected PRD body reaches state change. Medium fits only after AIDEV-04 is fixed |
| SEC-01 | High to Medium | Keep Medium | Skeptic reproduced it; it needs a valid login code in flight and a burst, so impact is bounded |
| PUB-01 | High to Medium | Keep Medium | R11 counts three addresses, not a credential |
| USER-03 | High to Medium | Keep Medium | Reproduced; a workaround exists |
| CLOSE-01, PM-01 | High to Medium | Keep Medium | Facts hold; fix once with TEST-01 |

I changed no severities.

## Root-cause clusters

| Cluster | Rows | Fix once |
|---|---|---|
| One client resolver | ARCH-01, ARCH-02 | A single `clientconfig.Resolve` used by CLI, `--api` and MCP |
| Session lifetime | USER-03, USER-04, FULL-02 | Persist and use the refresh token in apiclient |
| Help/version dispatch | USER-02, FULL-03, PM-04 | Pre-dispatch block before config load |
| Human-only boundary | AIDEV-04, FEAT-01, AIDEV-01 | Scoped agent token plus fenced tool results |
| Unwired guards and release gate | TEST-01, PM-01, PM-02, CLOSE-01, PUB-03 | CI schedule, `go test` and the four guards as release prerequisites |
| Wall clock | ARCH-04, QUAL-05 | Inject a clock in view builders; fixes the red L1 test |
| RLS write side | DATA-01, DATA-02, DATA-03, DATA-04 | Tighten the insert and update policies; add a write-side walk test |
| zh-first gaps | FEAT-05, COMP-01, COMP-02, COMP-03 | Rune counts, local-date handling, self-hosted assets |
| Abuse and cost | REL-01, SEC-02, COMP-04 | Per-user stream and input caps |
| Docs-truth drift | PM-06, COMP-08, README 14+ claim | Run the truth scripts in CI |

## Missed-findings table

| ID | Sev | File | Issue |
|---|---|---|---|
| COMP-01 | Medium | server/internal/api/ideas.go:92 | Byte-length limits reject long zh titles |
| COMP-02 | High | client/lib/chrome/time_ago.dart:3 | Deadline displays one day early for UTC+ |
| COMP-03 | Medium | client/web/index.html:25 | CDN-only fonts and CanvasKit |
| COMP-04 | Medium | server/internal/api/prds.go:114 | Unbounded sections go untruncated into prompts |
| COMP-05 | Medium | server/internal/store/project_repo.go:46 | Repo binding unverified; webhooks fan out by URL |
| COMP-06 | Low | client/android/app/src/main/AndroidManifest.xml:7 | Cleartext traffic allowed in the app manifest |
| COMP-07 | Low | scripts/backup-db.sh:29 | Plain-SQL dump with credential hashes, no umask |
| COMP-08 | Medium | PRODUCT.md:440 | Five PARTIAL promises untracked; completion needs no evidence |

## What this review could not cover

I made no network calls, so COMP-03 is unverified. I did not run the Flutter app on a device or emulator, did not run a screen reader, and did not test Postgres 14. I did not read the in-flight dirty checkout, per the brief. Skeptic repros were taken from repro-results.md and not re-run.

---

## Coordinator notes

**Score reconciliation.** Mean of the 13 reported scores 6.69; mean after the critic's reconciliation 6.24. The README score table carries both.

**Verdicts the critic would restore.** Severity changes only by skeptic verdict, so these stay as filed in the register and are listed for the owner:

| ID | Recommended | Reason |
|---|---|---|
| DATA-01 | High | R3 measured a non-owner self-inserting as admin and editing the workspace; the DB is the stated backstop and DATA-02 shows no test covers it |
| USER-02 | High | R6 measured exit 1 with a server-config error for help, version and typos; three reviewers filed the same root cause (FULL-03, PM-04), so Medium hides a front-door defect |
| AIDEV-01 | High | While AIDEV-04 stands (R5), an agent can accept its own proposal and commit, so an unfenced get_prd result reaches state change; Medium is right only after AIDEV-04 is fixed |

**Measured against verdict.** Three skeptic verdicts disagree with something the coordinator ran or the critic argued from a run:

| ID | Verdict | Measurement | Coordinator note |
|---|---|---|---|
| DATA-01 | Medium (was High): no API path reaches the INSERT | R3: as `insideout_app`, a bare `INSERT` of role `admin` into another user's workspace succeeds (0 to 1 visible workspaces, title `UPDATE` works) | The skeptic is right about reachability and the register keeps Medium. The hole is real at the SQL level and RLS exists to be the backstop when an API path is wrong, so plan row 2.6 sits in Wave 2, not Wave 3. |
| USER-02 | Medium (was High): the web app and docs show a path that avoids the CLI | R6: `--help`, `version` and a typo fail with `config: DATABASE_URL is required`, exit 1, with no config at all | Medium stands on reach (binary-only users), but it is a ~30-line fix on the first command anyone types, so it is plan row 1.2. |
| AIDEV-01 | Medium (was High) | R5: 52 MCP tools including `commit` and `proposal_decide` under one owner token | The critic would keep High while AIDEV-04 stands, since an injected PRD body then reaches a state change. The register keeps the skeptic verdict; plan row 2.5 removes the capability first and fences `get_prd` second. |

**[Verification] COMP-01 and COMP-02.** The critic filed COMP-02 as High after the skeptic stage, so the coordinator reproduced both instead of sending them to a skeptic (README R14 and R15). COMP-02: `TZ=Asia/Shanghai dart run` of the picker line and `isoDate` prints `sent=2026-10-17T16:00:00.000Z shown=2026-10-17` for a picked 2026-10-18; UTC and New York print the picked day. COMP-01: 67 Chinese characters are 201 bytes and `len(...) > 200` rejects them; a coordinator grep finds 16 string-length checks of this shape in `internal/api` (two of them cap usernames at 60 bytes; a 17th hit, `len(p.paths)`, counts a slice). COMP-03 is not verified (no network in the review).

**[Correction] PM-06.** The reviewer counted 76 broken relative links; the coordinator's three parsers count 74, 75 and 76. By the repo's own gate definition (markdown targets only) 4 are broken, all inside older review records; the other 71 point at source paths that moved or were deleted. "Zero broken doc links" holds for the gate's definition on its date and is false for a plain relative-link scan; the plan scopes the claim rather than counting it.

**Root-cause clusters.** One fix closes the rows in each cluster; plan rows are in [40-remediation-plan.md](40-remediation-plan.md).

| Cluster | Register IDs | Fix once | Plan row |
|---|---|---|---|
| C1 · client token and base plane | ARCH-01, ARCH-02, ARCH-05, FULL-02, USER-03, USER-04, USER-05, SEC-05, FULL-05 | one resolver (`clientconfig.Resolve`) plus a refresh path | 2.1, 2.2 |
| C2 · agent authority through MCP | AIDEV-04, FEAT-01, AIDEV-08, AIDEV-01 | server-enforced agent token scope; fence tool output | 2.5 |
| C3 · CI and release gate | PM-01, PM-02, PM-05, TEST-01, TEST-05, CLOSE-01, PUB-03, PUB-07, PUB-08, SEC-07 | a schedule, a release gate, and the five guards in CI | 1.4, 1.5, 1.6, 3.4 |
| C4 · wall clock | ARCH-04, QUAL-05 | inject `Server.now`; guard test against raw `time.Now()` | 1.3 |
| C5 · first run | USER-01, USER-02, USER-06, USER-07, USER-09, USER-10, FULL-03, PM-04, PM-08 | first-admin bootstrap; help and version before config; copy fixes | 1.1, 1.2, 1.7, 1.10, 3.8 |
| C6 · proposal apply and expand | FEAT-02, FEAT-09, FULL-01 | `applied_at` plus one transaction; client reads `children` and confirms | 2.3, 2.4 |
| C7 · RLS write side | DATA-01, DATA-02, DATA-03, DATA-04, DATA-08 | one migration, column grants, write-side walk, refuse BYPASSRLS | 2.6 |
| C8 · privacy and public hygiene | PUB-01, PUB-02, PUB-04, PUB-05, PUB-06, PUB-10, PUB-11, PUB-12, PM-11 | scrub plus a redaction rule; owner decisions on history and keys | 0.1 to 0.4, 3.5 |
| C9 · coach gate honesty | AIDEV-02, AIDEV-03, AIDEV-05, AIDEV-06, AIDEV-07, CLOSE-03 | surface skips, persist declines, honour truncation | 3.1 |
| C10 · stream and ops bounds | REL-01, REL-02, REL-03, REL-04, REL-05, SEC-08, FULL-04 | stream caps, deadlines, grace period | 1.8, 3.2 |
| C11 · docs truth | ARCH-06, PM-03, PM-06, PM-07, PM-09, CLOSE-04, USER-08 | make the architecture and ledger records true; scope the claims | 3.6 |
| C12 · login abuse | SEC-01, SEC-02, SEC-03, SEC-04, REL-07 | atomic attempts, trusted-proxy hops, async mail | 1.9, 3.3 |
| critic: One client resolver | ARCH-01, ARCH-02 | A single clientconfig.Resolve used by CLI, --api flags and the MCP binary | see plan |
| critic: Session lifetime | USER-03, USER-04, FULL-02 | Persist and use the refresh token in apiclient | see plan |
| critic: Help/version dispatch | USER-02, FULL-03, PM-04 | Pre-dispatch block in main.go before config load | see plan |
| critic: Human-only boundary | AIDEV-04, FEAT-01, AIDEV-01 | Scoped agent token plus fenced tool results | see plan |
| critic: Unwired guards and release gate | TEST-01, PM-01, PM-02, CLOSE-01, PUB-03 | CI schedule, go test and the four guards as release prerequisites | see plan |
| critic: Wall clock | ARCH-04, QUAL-05 | Inject a clock into view builders; also fixes the red L1 test | see plan |
| critic: RLS write side | DATA-01, DATA-02, DATA-03, DATA-04 | Tighten insert and update policies and add a write-side walk test | see plan |
| critic: zh-first gaps | FEAT-05, COMP-01, COMP-02, COMP-03 | Rune counts, local-date handling, self-hosted assets | see plan |
| critic: Abuse and cost | REL-01, SEC-02, COMP-04 | Per-user stream and input caps | see plan |
| critic: Docs-truth drift | PM-06, COMP-08 | Run the truth scripts in CI and make the promise guard fail on untracked PARTIAL rows | see plan |

**What this review could not cover.**

- No network access, so COMP-03 (CDN reachability from mainland China) is unverified
- Flutter app not run on a device, emulator or screen reader; accessibility is unreviewed
- PostgreSQL 14 not tested although docs claim 14+
- Skeptic and coordinator reproductions were taken from repro-results.md and not re-run
- The in-flight dirty checkout was not read, per the brief
- Presence cursor server-side rate limiting was only read, not load-tested

## What this role would do first

Fix COMP-02 (deadline date) and COMP-01 (rune counts) in one change with two tests, then restore DATA-01 and USER-02 to High in the ledger.
