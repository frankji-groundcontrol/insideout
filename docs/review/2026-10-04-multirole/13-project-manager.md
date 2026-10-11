# 13 · Project Manager

**Lens:** stranger onboarding via README, docs record-system currency, README↔PRODUCT.md consistency, CI/ownership/release gating, claimed-vs-deployed, license/board hygiene.
**Score:** **5.0 / 10**
**Bottom line:** The repository has an unusually disciplined records culture — dated changelogs, an authoritative task board, a handoff file — but the three documents a newcomer touches first all mislead: the README's docker quick start cannot run, its "core features" sell the product model PRODUCT.md has superseded, and the HANDOFF/task board still say "merge PR #3" sixteen days after that merge, with no evidence the accepted Cobalt build ever reached the hosted deployment. Nothing gates merges (no CI, no tags, no CODEOWNERS), and what production serves is unverifiable from the repo. The project works for its sole maintainer on the `dev.sh` path, but as a documented, releasable project it is drifting.

Reviewed against main@877ad03; High/Critical findings re-checked by an independent skeptic — verdicts in [01-findings-register.md](01-findings-register.md).

---
## Scope

README.md (both languages), PRODUCT.md, docs/PLAN.md, docs/TODO.md, docs/plans/README.md, docs/HANDOFF.md, docs/changelogs/README.md, docs/issues/README.md, LICENSE, `git log --all --oneline | head -40`, plus spot checks of docker-compose.yml, docs/index.md, and the hosted deployments' served artifacts (no secrets touched). Verification run: `docker compose config` (fails, exit 15), `git tag`, `git ls-files | grep -i codeowners`, curl of the hosted app.

## What is good

- **The records system is real and current where it counts.** docs/changelogs/README.md runs from 2026-09-18 back to the 2026-07-20 Go rewrite, each entry stating what changed and how it was verified; docs/issues/README.md keeps numbered bug records (BUG-001…BUG-015) with root causes.
- **Status semantics are written down.** docs/plans/README.md:36-40 defines In flight / Pending / Finished / Completed / Blocked and warns against `git add .` because tasks share documentation files.
- **README env hygiene is honest.** README.md:69 names exactly the two required variables and the offline coach fallback; README.md:66-67 explains *why* `dev.sh` is mandatory (the app reads the process environment, not `.env`).
- **Font licensing is handled well.** README.md:96 distinguishes "free to use" from redistributable and states no font binaries ship (PuHuiTi section marked historical, README.md:100).
- **The deploy gotcha is recorded.** docs/HANDOFF.md:55-60 documents the nginx stale-upstream 504 after a `server` redeploy and gives the correct probe — knowledge that usually dies with a maintainer.

## Findings

### PM-01 · High · README quick start is broken: `docker compose up -d` cannot run
- **File:** README.md:72-77 (guard at docker-compose.yml:17-18)
- **Evidence:** README.md:72-77: "Or bring up the whole stack with docker-compose: … `docker compose up -d`" — while docker-compose.yml:17 is `INSIDEOUT_OWNER_PASSWORD: ${POSTGRES_OWNER_PASSWORD:?set POSTGRES_OWNER_PASSWORD in .env}`. I ran `docker compose config`: exit 15, "required variable POSTGRES_OWNER_PASSWORD is missing a value".
- **Failure scenario:** A stranger sets the two variables README.md:69 calls required, runs the documented `docker compose up -d` → compose refuses to parse, even when `DATABASE_URL` targets a remote instance and the bundled postgres is unused (docker-compose.yml:2-4: "Not used when DATABASE_URL points at an external… instance"). Compose also ships no Flutter client, so "whole stack" overpromises regardless.
- **Fix:** Replace the `:?` guards on the two postgres-local passwords with `:-` defaults (they are only needed when compose's own postgres is the target) and correct "whole stack" to "backend stack (postgres + server)". ~4 lines + one README line.

### PM-02 · High · HANDOFF and task board still say "merge PR #3" — 16 days after the merge — and production state is unverifiable
- **File:** docs/HANDOFF.md:10-11; docs/plans/README.md:13-14
- **Evidence:** docs/HANDOFF.md:10-11: "1. Next: merge [PR #3](https://github.com/<org>/insideout/pull/3) (Cobalt restyle + live-API wiring), then `railway up --service app`." — but `git log` shows "877ad03 Merge pull request #3 … 2026-09-18". docs/plans/README.md:13 lists the same task as "**Finished** | Merge [PR #3]…, then `railway up --service app`." The walk record (docs/design-qa/2026-09-18-cobalt-backend-walk.md:25) says "this walk did not open a hosted session"; the plan (docs/plans/2026-09-18-cobalt-backend.md:28) defers deploy "unless the user asked to ship". No record after 2026-09-18 shows any `railway up`; autodeploy is off (docs/HANDOFF.md:52).
- **Failure scenario:** A returning agent resumes from HANDOFF, tries to "merge PR #3" — already merged — then cannot tell whether Cobalt is live. I probed the hosted app: its Flutter bundle contains zero `cobalt` occurrences, consistent with (not proof of) a pre-Cobalt build. The deploy never ran, or ran unrecorded; the record system cannot say what production serves.
- **Fix:** Deploy (`railway up --service app`), verify Cobalt on `<prod-domain>`, then flip both board rows and HANDOFF item 1 to "deployed and verified" with the date. ~10 lines across three files after the live check.

### PM-03 · Medium · README "core features" market the product model PRODUCT.md has superseded
- **File:** README.md:28-33
- **Evidence:** README.md:29: "an AI coach walks you through four stages: **clarify** … → **finalize**"; README.md:32-33: "a workspace admin approves or rejects it". PRODUCT.md:418-420: "This baseline supersedes the earlier product assumptions that … every coaching session follows four rigid stages, a Workspace admin approves all PRDs".
- **Failure scenario:** A new user or contributor reads the README and expects — or critiques the product against — a rigid four-stage coach and admin gate, exactly what PRODUCT.md says the product is leaving. The feature list itself carries no marker, so the superseded model is what gets quoted.
- **Fix:** Add one line under "核心功能" noting the section describes the *current build* against the PRODUCT.md target (its own PRODUCT.md:424-428 "running foundation" carve-out). ~2 lines.

### PM-04 · Medium · TODO.md is a two-month-stale tracker for a deleted frontend, still linked as the "Progress tracker"
- **File:** docs/TODO.md (README.md:84)
- **Evidence:** README.md:84: "开发进度 / Progress tracker: [`docs/TODO.md`](docs/TODO.md)". docs/TODO.md:7: "**Last updated**: 2026-08-13"; its completed sections describe the Nuxt app deleted 2026-08-18 (docs/TODO.md:94 "Nitro same-origin proxy", :149 "Prisma cinematic re-theme", :143 "`GithubSync.vue`").
- **Failure scenario:** The stranger opens the linked "progress tracker" and learns project status from a frontend that no longer exists, while real progress since 2026-08-13 (CLI/MCP parity, email auth, invitation signup, Cobalt) is invisible outside the changelog index. Two disagreeing "progress" sources beat none only when labeled.
- **Fix:** Refresh TODO.md's delta to the current baseline and mark P5-P8 historical, or retire it into docs/history/ and repoint README.md:84 at docs/plans/README.md. ~15 lines.

### PM-05 · Medium · No CI, no tags, no CODEOWNERS — quality and release claims are unverifiable by anyone but the author
- **File:** repo-wide (claim site docs/plans/README.md:14)
- **Evidence:** `.github/` does not exist in the tracked tree, `git tag` is empty, and no CODEOWNERS/OWNERS file is tracked. Yet docs/plans/README.md:14 asserts: "Analyze + 54 tests green; walk recorded."
- **Failure scenario:** With autodeploy off and no CI, "green" is a memory claim. PR #3's four client commits merged with no external check, and PM-02 shows the follow-through went unverified too. Nothing marks main@877ad03 as a releasable point.
- **Fix:** Add one minimal workflow (`go build/vet/test` on pull_request) and tag the head. ~30 lines of YAML.

### PM-06 · Medium · PRODUCT.md brand commitment contradicts the accepted, shipped Cobalt identity — no supersession record
- **File:** PRODUCT.md:435-437
- **Evidence:** PRODUCT.md:436-437: "Visual identity: the existing Ink & Seal system—celadon ground, sumi-ink text, a single vermilion accent, Noto Serif SC display…". docs/index.md:49-50: "live Flutter is Cobalt **1.0.0**; lock screens in [cobalt/](design-system/cobalt/README.md)".
- **Failure scenario:** PRODUCT.md declares itself the winning authority (PRODUCT.md:6-9), and its brand commitment is unamended since 2026-08-13. A future contributor following it will "restore Ink & Seal" and undo the accepted Cobalt lock — which is literally what happened on 2026-08-19 before Cobalt was accepted.
- **Fix:** Update Brand commitments to Cobalt, or add a supersession note pointing at docs/design-system/cobalt/ and the 2026-09-17 direction record. ~3 lines.

### PM-07 · Low · HANDOFF internal accuracy: numbering runs 1,2,2,4,5; "27 MCP tools" contradicts the 50-tool record
- **File:** docs/HANDOFF.md:16,22,24,29,37
- **Evidence:** docs/HANDOFF.md:22: "the full parity loop (27 MCP tools)"; docs/changelogs/README.md:32 records "46 → 50 tools"; docs/plans/README.md:16 agrees. The list reads "2." at :16 and again at :24, then "4.", "5.".
- **Failure scenario:** Anyone citing HANDOFF for the tool surface propagates a count 23 tools out of date — the same drift class as PM-02, one level smaller.
- **Fix:** Renumber; update the count. 2 lines.

### PM-08 · Low · Deployment identifiers committed in tracked docs violate the repo's own redaction rule
- **File:** README.md:64; docs/HANDOFF.md:42-44
- **Evidence:** README.md:64: "托管站点 / hosted: `https://<railway-host>`"; docs/HANDOFF.md:43 embeds `API_BASE=https://<prod-domain>/api/v1`; at least nine more docs match. AGENTS.md requires: "Redact provider/project identifiers, database hostnames, workspace IDs, personal emails, and local runtime paths in docs and examples."
- **Failure scenario:** The repo is already public; the rule exists so hosts stay un enumerable and records portable. Every future doc copies these, deepening the exposure.
- **Fix:** Decide once whether deployment URLs are exempt (then amend AGENTS.md) or sweep them to placeholders. Mechanical sweep, ~12 files.

### PM-09 · Low · MPL-2.0 present and consistent, but the README never says so
- **File:** LICENSE (absence at README.md:79-84)
- **Evidence:** LICENSE:1 "Mozilla Public License Version 2.0"; the README's "Learn More" list (README.md:79-84) has no License entry.
- **Failure scenario:** A potential user or contributor cannot tell from the entry page that the code is weak-copyleft; some will assume all-rights-reserved and walk away.- **Fix:** Add a one-line License bullet under "Learn More". 1 line.

## Lead verification

| Lead | Verdict | Evidence |
| --- | --- | --- |
| L1 (no CI/tags/ownership) | **Confirmed** | `.github/` absent from tracked tree; `git tag` empty; no CODEOWNERS via `git ls-files` |
| L2 (compose fails; README "whole stack" overpromise) | **Confirmed** | `docker compose config` exit 15 on POSTGRES_OWNER_PASSWORD; docker-compose.yml:17-18 `:?` guards; compose is postgres+server only |
| L3 (README describes superseded model) | **Confirmed** | README.md:29,32-33 (four stages; admin approves) vs PRODUCT.md:418-420 |
| L4 (PRODUCT.md brand vs shipped Cobalt) | **Confirmed, nuance** | PRODUCT.md:436-437 mandates Ink & Seal; docs/index.md:49-50 says live Flutter is Cobalt 1.0.0 — the canonical doc is the stale side |
| L5 (HANDOFF/board stale) | **Confirmed** | HANDOFF.md:10 "Next: merge PR #3" vs merge commit 877ad03 dated 2026-09-18; numbering 1,2,2,4,5; "27 MCP tools" (HANDOFF.md:22) vs "46 → 50 tools" (changelogs README:32) |
| L11 (identifiers in tracked docs) | **Confirmed** | README.md:64, HANDOFF.md:43, plus 9 more grep-matching files |
| L14 (deployed behind main) | **Confirmed** | No deploy record after the 2026-09-18 merge; walk record "did not open a hosted session"; hosted bundle has 0 `cobalt` occurrences (weak alone, decisive with the missing record) |
| L12 (Go build/vet/test clean) | **Accepted as reported** | Not re-run in this lens; consistent with docs/PLAN.md:62-65 conventions |
| L13 (flutter analyze/test green) | **Not run** | Out of PM lens; the 2026-09-18 claim is unrecheckable without automation (PM-05) |

## Recommendations

1. Fix the compose interpolation guards so `docker compose config` parses with only `DATABASE_URL` + `INSIDEOUT_JWT_SECRET` set; correct README's "whole stack" wording (PM-01).
2. Deploy main@877ad03's app service, verify Cobalt on `<prod-domain>`, and only then flip the two board rows and HANDOFF item 1 to "deployed and verified" (PM-02).
3. Mark README's Core Features as the current-build baseline against the PRODUCT.md target (PM-03).
4. Refresh or retire docs/TODO.md and repoint README.md:84 (PM-04).
5. Add a minimal CI workflow (build/vet/test) and tag the current head (PM-05).
6. Amend PRODUCT.md's Brand commitments to Cobalt with a supersession pointer (PM-06).
7. Fix HANDOFF numbering and the MCP tool count; fold the redaction sweep and License bullet into one docs checkpoint (PM-07, PM-08, PM-09).

## What this role would do first

Stand up the missing release gate: run `railway up --service app` against main@877ad03, confirm the Cobalt build on `<prod-domain>` with a signed-in page load, and only then update docs/HANDOFF.md and the two plans-board rows to "deployed and verified 2026-10-04" — every record is currently anchored to a "next: merge PR #3" world that stopped existing sixteen days ago, and no documentation work matters until the record system re-establishes what production actually serves.
