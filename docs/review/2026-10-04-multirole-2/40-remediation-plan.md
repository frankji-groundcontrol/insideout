# Remediation plan — multirole review 2

Waves ordered by impact over effort. Register IDs refer to
[01-findings-register.md](01-findings-register.md). Repo rules apply throughout:
DB-gated tests against real PostgreSQL, failing test first for substantial
behavior changes, SSE changes exercised against a live server before "done".

## Wave 0 — secrets, branch, license, ownership (verified clean)

Nothing to remediate. No key-shaped strings in the tree (both scans clean), the
default branch is the product, the repo is private with no license finding.
One standing **user action** carried from the first review, not code: rotate the
Resend mail key if any local transcript ever leaves the machine.

**Check:** the two `rg` scans pasted in this review's changelog, `git status` clean.

## Wave 1 — one-line fixes, one commit

| Step | IDs | File | Change | Size |
|---|---|---|---|---|
| Capture `window.open` null; make fallback live | FS-01, CQ-02 | `client/lib/api/download_web.dart:19` | `final w = web.window.open(url, '_blank'); if (w == null) return false;` + one `feature_wiring` substring on `export_page.dart` | ~3 lines |
| Pin CI to the artifact toolchain | ARCH-01 | `.github/workflows/ci.yml:81` | `flutter-version: 3.44.0` to match `client/Dockerfile:3` (upgrade deliberately later) | 1 line |
| Widen the skip-guard to the whole suite | TEST-09 | `.github/workflows/ci.yml:64` | grep `--- SKIP` over the single verbose `go test ./...` run (every current skip is a DB gate); drop the duplicate store-only re-run | ~4 lines |
| Guarded date truncation everywhere | CQ-03, CQ-04 | `chrome/time_ago.dart`, `roadmap_node_card.dart:53,164`, `roadmap_progress_view.dart:104` | expose the guarded `iso` helper; route all three `substring(0, 10)` sites through it | ~6 lines |
| Name the revoke delay | CQ-05 | `client/lib/api/download_web.dart:22` | `const _revokeDelayMs = 60_000;` | 2 lines |
| Record the deploy | PM-13 | `docs/HANDOFF.md`, board row | append commit+date+verify result for the 26db979 deploy (done, unrecorded) | docs |
| Name the running authority model | CC-02 | `PRODUCT.md` status section | one sentence: admin-approves-PRDs is the current v1 path, superseded model is the target | docs |

**Check:** `go test ./... -count=1` (0 skips), `flutter test`, CI green including
`flutter build web`.

## Wave 2 — confirmed-High fixes, failing tests first

| Step | IDs | File | Change | Size |
|---|---|---|---|---|
| Critic gets a message payload | AIDEV-09 | `server/internal/agent/critic.go:231` | pass `[]Message{{Role: RoleUser, Content: …call report_findings}}` like the planner; add a wire-level test asserting forced-tool requests never marshal empty `messages`; one live exercise per the streaming practice | ~20 lines |
| Drop visibly drops | USER-09 | `server/internal/store/ideas.go`, `ideas_page.dart` | exclude `status = 'dropped'` in `ListIdeas` (DB-gated test first: author's list omits dropped), confirm dialog via the existing `idea.dropConfirm` key, localize the dropped state | ~30 lines |
| SSE 401 refresh on web | USER-10 | `client/lib/api/client.dart:450`, `sse_transport_web.dart` | on 401, await the existing single-flight refresh, retry once with the rotated token, else `clearTokens` so the router lands on the auth door | ~40 lines |
| l10n error mapper | USER-11 | new small `client/lib/api/errors_l10n.dart` + 12 call sites | map `ApiException` code/status → zh key with generic fallback; replace every `Text('${snap.error}')` | ~60 lines |
| Proposal web loop | FEAT-03 | `project_page.dart:125` | special-case `kind == 'agent_proposal'`: Accept/Reject actions calling the existing decision endpoint; hide edit/delete for that kind | ~60 lines |

**Check:** `go test ./... -count=1 -race` (0 skips), `flutter test` + new widget
tests, live SSE exercise on a local server, then deploy both services and
domain-verify per ship loop.

## Wave 3 — hardening and the two reliability decisions

| Step | IDs | File | Change | Size |
|---|---|---|---|---|
| Trusted-proxy depth | REL-13, SEC-10 | `auth_throttle.go:292` | first run the canary: observe the XFF chain the server receives through the domain; then prefer `X-Real-IP` set by nginx or make rightmost-count configurable; add a two-hop unit test | canary + ~15 lines |
| Band scaffold extraction | CQ-01 | new `roadmap_band_scaffold.dart` | shared label/chrome scaffold for `_BandRow` and `_ProgressBand`; both views pass item cards | ~80 lines moved |
| Backup story | CC-03 | `docs/usage/`, maybe `scripts/` | check whether Railway Postgres backups are enabled; either reference them in deployment docs or add a nightly `pg_dump` script + tested restore doc | ~15 lines + docs |
| Deployment events: decide | CC-01 | `github_webhook.go` or docs | either map `deployment` into the evidence pipeline with a release kind, or remove it from the guide template and PRODUCT.md until built | ~20 lines or docs |

**Check:** two-hop throttle test green; `flutter test` after scaffold extraction;
restore doc walked once against a scratch database.

## Wave 4 — features left to implement (slice-sized, in order)

The explicitly-future ledger the owner asked for. Each slice lands behind the
wave-2 fixes. PRODUCT.md's end-state loop is the source of truth.

| Slice | IDs | First increment |
|---|---|---|
| Decision Log, smallest slice | FEAT-02, PM-10 | `decisionNote` field in `VersionsCommitDialog` + render in the commit detail (server already accepts it) |
| Product Branches, slice 1 | FEAT-01, PM-11 | `prd_branches` table (source commit, title, driver) + `POST /prds/{id}/branch` + branch badge in versions stack |
| Per-project authority, slice 1 | PM-12 | `project_members` with the five roles; creator → Driver; gate commit confirmation on Driver + Decision Owner |
| Then the Medium backlog | FEAT-04, FEAT-06..12, PM-14..18, USER-12..17 | evidence read UI, load-more, lenses, personas, adaptive coach, overdue-reason, feedback invitations, exceptions-first overview |
| Release hygiene | PM-19, PM-02 | tag a version, CODEOWNERS, and the human signed-in hosted page load |

**Check per slice:** DB-gated test first, one dated plan in `docs/plans/`,
changelog on merge.

## Not in scope for this release

- Full Merge flow (needs Branch slice 1 first) and the full authority rollout.
- Flutter toolchain upgrade (Wave 1 only pins CI down; upgrade is its own change).
- Pagination redesign behind load-more; load/concurrency testing; `-race` beyond CI.
- Mobile/desktop Flutter builds; mail deliverability hardening (SPF/DKIM).
- Splitting `en.dart` or `roadmap_test.go` — inside the rule's intent; record the
  exemption once in the modularity note instead.
- Rotating the Resend key — user action, recorded, not code.
