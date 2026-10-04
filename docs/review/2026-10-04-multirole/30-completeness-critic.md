# 30 · Completeness critic

**Role:** close the 10-reviewer audit — verify coverage against the advertised briefs, hunt claims no role filed, check scores against their own findings, and re-run the coordinator's measurements.

Reviewed against main@877ad03; critic re-ran the coordinator's measurements independently.

---

## 1. Dimension coverage: none missing

All ten advertised briefs were executed with original in-scope findings: 10-architect (ARCH-01..11), 11-ai-developer (AIDEV-01..08), 12-fullstack (FS-01..11), 13-project-manager (PM-01..09), 14-user (USER-01..08), 20-feature-implementation (FEAT-01..07), 21-code-quality (CQ-01..06), 22-security (SEC-01..09), 23-testing (TEST-01..08), 24-reliability-ops (REL-01..12). No orphan brief exists. The real gaps are unbriefed dimensions: accessibility, performance/load, native mobile builds, and live-provider behavior (see §6).

## 2. The unfiled README contradiction: fonts

Every candidate claim checks out as filed or true — *except one*. README.md:89 says "repository ships no font files" and README.md:99 says "**This repository contains and distributes no font files.**" Both are false: `client/assets/fonts/` holds NotoSansSC-Var.ttf (17.8 MB) and NotoSerifSC-Var.ttf (25.1 MB), bundled at `client/pubspec.yaml:66` (`- assets/fonts/`) and loaded by `client/lib/theme/native_fonts.dart:14-15`. No register entry named this claim. Worse, the OFL binaries ship without any OFL license text (grep matches only README.md and a 2026-07-27 changelog), so the repo redistributes OFL fonts without the required notice while claiming it distributes none. Filed as CRIT-01.

Cleared on inspection: "real-time" board → FEAT-02; print-to-PDF → FEAT-03; README.md:64 hosted link → PM-08/SEC-08; quick-start ordering → USER-01; tool counts → PM-07; bilingual parity is exact (319 keys each way) so no finding; README.md:70's "only two uncommented in .env.example" is **true** (exactly 2); no "five auth ways" claim exists in README/PRODUCT.

## 3. Score calibration: the two 7.2s disagree with each other

11-ai-developer and 21-code-quality both score **7.2**, but their severity profiles are incompatible. AI-developer owns a CONFIRMED High (AIDEV-01: retryStreamChat concatenates a full second answer onto a partial stream and persists divergent history), a Medium its own file calls happy-path-breaking (AIDEV-02: 20-turn quote-grounding window "denies the documented clarify→draft progression outright"), and a gate-honesty hole (AIDEV-03: offline finalize passes with zero critic review). Code-quality owns zero Highs — its worst are bookkeeping (CQ-01 untracked oversize files, CQ-02 substring tests). Severity-weighted, ai-developer belongs below architect (6.0); identical scores here mean at least one is miscalibrated. **Adjustment applied in the README and executive summary: ai-developer 7.2 → 5.8.** Secondary: security at 7.0 despite SEC-01 being an unauthenticated, remotely exploitable brute-force/DoS on the hosted instance today ("brute-forceable at unlimited speed", 22-security.md:69: the hosted API "must not be exposed past localhost until SEC-01 and SEC-02 land") — scored above architect and feature-implementation whose Highs are docs/UI overpromise, not live holes. The converse pattern does not occur: every 5.x role owns CONFIRMED Highs, so the low scores are consistent with their files.

## 4. Verdict vs coordinator measurement

**TEST-02 census.** Register and the testing role say "43 of 188 Go test results skip"; the coordinator reported "290 results / 43 SKIP". Critic recount: `go test ./... -count=1 -v` yields **145 top-level PASS + 43 SKIP = 188** top-level results, and **247 total `=== RUN` lines** — Go 1.25 prints subtest runs at column 0 (145+59 subtests+43=247). 290 matches neither; it is 247+43, a double-count. The CONFIRMED verdict (43 skips never execute; no CI) survives both numbers; the reproducible figure is **188 top-level / 43 SKIP**.

**REL-02 rationale.** The downgrade to Medium ("dev-only by design") is contradicted by the coordinator-confirmed compose measurement: docker-compose.yml:38-52 forwards no `INSIDEOUT_MAIL_API_KEY`/`PUBLIC_URL`, so dev-mode mail — which logs live 15-minute sign-in URLs at Info — is the *default on the documented self-host path*, not a local-only choice. Medium severity still defensible; the rationale is not. The other downgrade checks out exactly (.env.example:131-132 are commented `#INSIDEOUT_GH_CLIENT_ID=` declarations per env_catalog.py:13; both test invocations reproduce 46 passed/1 failed; `docker compose config` exits 15 here too).

## 5. Missed findings (verified by opening each file)

| ID | Sev | File:line | Summary |
|---|---|---|---|
| CRIT-01 | Medium | README.md:88-99 vs client/assets/fonts/ | "Ships no font files" false (two Noto .ttf, ~43 MB, bundled at pubspec.yaml:66, loaded at native_fonts.dart:14-15); no OFL text ships with them |
| CRIT-02 | Low | SKILL.md:19 | Production API domain hardcoded with copy-paste `export INSIDEOUT_API=...` instructions; absent from every redaction finding's evidence list |
| CRIT-03 | Low | client/Dockerfile:12-17 | Hosted client image: nginx master as root, no USER, no HEALTHCHECK — FS-07/SEC-03 filed this class only for the server image |
| CRIT-04 | Low | docs/plans/README.md:12 | Review's own board row links to `docs/review/2026-10-04-multirole/README.md`, which did not exist while the review ran (closed by this folder's README) |
| CRIT-05 | Low | CLAUDE.md:14-15 | Router drift vs AGENTS.md:5 parity claim; CLAUDE.md still routes agents to stale docs/TODO.md "for known limitations" |

CRIT-01 fix: rewrite the Third-party assets section to state the repo ships OFL Noto Sans/Serif SC variable fonts for native targets and references PuHuiTi by name only; add an OFL.txt alongside the fonts. First check: confirm web builds never bundle the assets (client/lib/theme/web_fonts_web.dart loads Google Fonts).

## 6. What this review could not cover

Production ground truth (no Railway access — PM-02/REL-12 rest on records plus a 0-`cobalt`-occurrence bundle probe); any `.env`/secret value, by design; execution of the 43 skipped DB-integration tests — *no reviewer ran them*, so RLS/authz was verified by reading SQL, not running it; live LLM provider calls; a browser walk of the Cobalt UI; iOS/Android builds (now material, given CRIT-01's ~43 MB of native-only font assets); and unbriefed dimensions — accessibility, performance/load, subagent-config hygiene (.claude/.trae/.sisyphus/.gstack).

Mean score 6.2 is fair; the doorway is broken (USER-01, FS-02/03, ARCH-01) while the engine underneath is genuinely sound — and this review's own board row (CRIT-04) proves the records rot is contagious.
