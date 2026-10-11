# Multirole review 3 (2026-10-10): what happened to each finding

The register is [2026-10-10-multirole-3/01-findings-register.md](../2026-10-10-multirole-3/01-findings-register.md) (83 rows: 0 Critical · 17 High · 33 Medium · 33 Low). Unlike the earlier registers, review 3 shipped a [remediation plan](../2026-10-10-multirole-3/40-remediation-plan.md) that was then executed as four verified waves, so every row below is traced to its wave, red test and fix commit. [Reading the statuses](README.md#statuses).

## Method

Four dynamic-workflow waves (32 clusters) executed the plan. Each cluster worked in its own worktree on a red-test-first contract; an independent adversarial verifier reproduced the red test, the green, re-ran the full gates and mutation-checked the fix before acceptance — all 32 clusters were accepted in round 1. Three rows the workers could not close inside their file ownership (CQ-03, TEST-06, USER-08) were closed by the integrator across the cluster boundaries; CQ-04 was parked by its brief's own pure-move gate rather than silently dropped. Integrated on `main` (ranges include the between-wave record commits):

- wave 1 — `5c8f54f..c89faea` (gate, toolchain, planner, sse, stage, email, idea)
- wave 2 — `c89faea..ce8e173` (client guards, store bounds, turn bounds, mail signal, provider, mcp, throttle, urls, github, honest degradation, ledger caps)
- wave 3 — `ce8e173..b056e30` (client polish ×2, docs truth, env catalog, prompts contract, test hygiene)
- wave 4 — `b056e30..4464330` (locale threading, validation slice, promise register, gap issues, upgrade lane, hosted smoke, contribution surface, attestation honesty)

What each fix does, the operator migration notes and the final gate run live in the [changelog folder](../../changelogs/2026-10-11-rv3-remediation/index.md). Six rows remain **Owner** — hosted deploy, mail/LLM keys, or the first `v*` tag — enumerated in [HANDOFF](../../HANDOFF.md) and the [promise register](../2026-10-10-multirole-3/50-promise-register.md); they are the only rows not closed by code.

| ID | Sev | Status | Resolution |
|---|---|---|---|
| AIDEV-01 | H | Fixed | wave 1 — fix ec60e5d · red TestCaptureRankedCritique_RestatementCannotEraseStoredFindin… |
| AIDEV-02 | H | Fixed | wave 1 — fix fde5df0 · red TestExpandNode_FencesUntrustedNodeText |
| AIDEV-03 | M | Fixed | wave 2 — fix 953977daaf · red server/internal/agent/taxonomy_band_test.go |
| AIDEV-04 | M | Fixed | wave 2 — fix bb19273 · red TestCommitToolSchemaDeclaresUnresolved |
| AIDEV-05 | M | Fixed | wave 2 — fix 159ea69 · red TestHandleMessage_CompletionRequiresTheMarkerAsTheFinalLine/… |
| AIDEV-06 | M | Fixed | wave 2 — fix 446aee6400 · red server/internal/agent/responses_request_test.go |
| AIDEV-07 | M | Fixed | wave 1 — Fixed in wave 1 + integrator seam: stage confirmation applies parked resolutions atomically (store.MetaTransform + agent.ApplyStageConfirmation) with a decision trail |
| AIDEV-08 | M | Fixed | wave 3 — fix 50999ff · red TestSystemPromptClarifyFollowsCoachingContract |
| AIDEV-09 | M | Fixed | wave 4 — fix aa9bd9685d |
| AIDEV-10 | L | Fixed | wave 2 — fix 57cec5d4c2 · red server/internal/agent/ledger_caps_test.go |
| AIDEV-11 | L | Fixed | wave 2 — fix c1c220e · red TestHandleMessage_TotalTurnDeadlineCutsATricklingTurn |
| AIDEV-12 | L | Fixed | wave 3 — fix 9132ddc · red TestParseRankedCritique_DropsUnknownSection |
| ARCH-01 | H | Fixed | wave 1 — fix b16b5b3 · red scripts/tests/check_go_floor.py |
| ARCH-02 | L | Fixed | wave 1 — fix 480d85b |
| ARCH-03 | M | Fixed | wave 2 — fix dc1d665 · red server/internal/api/admin_mail_probe_test.go |
| ARCH-04 | M | Fixed | wave 3 — fix cbc925cab5 · red check_docs_truth.py |
| ARCH-05 | L | Fixed | wave 3 — fix 27baac002b · red check_docs_truth.py |
| COMP-01 | H | Fixed | wave 4 — fix 1c26951 · red docs/review/2026-10-10-multirole-3/checks/promise-register.s… |
| COMP-02 | M | Fixed | wave 3 — fix efc16d4ece · red check_docs_truth.py |
| COMP-04 | L | Fixed | wave 3 — fix 9f76c929a8 · red check_docs_truth.py |
| CQ-01 | M | Fixed | wave 3 — fix 34eef9d · red client/test/auth_door_spacing_scan_test.dart |
| CQ-02 | M | Fixed | wave 4 — fix 5af2c94 |
| CQ-03 | M | Fixed | wave 2 — Fixed by the integrator after the cluster's pure-move gate correctly refused: one showSnapshotRevision flow serves both pages |
| CQ-05 | L | Fixed | wave 3 — fix 0d55bda |
| CQ-06 | L | Fixed | wave 3 — fix f209560 |
| CQ-07 | L | Fixed | wave 2 — fix 2344973 |
| FEAT-01 | H | Fixed | wave 4 — fix 4d79db1 · red server/internal/api/conversations_locale_test.go |
| FEAT-02 | H | Fixed | wave 1 — fix a5e8bc0 · red TestConvertIdea_PreservesBodyAsFirstMessage |
| FEAT-04 | H | Fixed | wave 4 — fix 86c15a0 |
| FEAT-05 | M | Fixed | wave 4 — fix 3bd86f0 |
| FEAT-06 | M | Fixed | wave 4 — fix 71d884d |
| FEAT-07 | L | Fixed | wave 2 — fix a98e298a67 · red TestPlanGuideEvidenceDeploymentStatusFailure |
| FEAT-08 | L | Fixed | wave 3 — fix 27baac002b |
| FULL-01 | H | Fixed | wave 1 — fix 83484be · red TestInvitationEmailExpiryCopyAgreesAboveADay |
| FULL-02 | M | Fixed | wave 2 — fix 5aba17f · red client/test/prd_page_load_dispose_test.dart |
| FULL-03 | M | Fixed | wave 4 — fix d4598ac |
| FULL-04 | L | Fixed | wave 3 — fix be9f428 · red client/test/dock_dead_taps_test.dart |
| FULL-05 | L | Fixed | wave 3 — fix e158aeace6 · red client/test/export_page_title_test.dart |
| FULL-06 | L | Fixed | wave 1 — fix 01fa61b · red TestSimpleEmailHTMLButtonIsCobaltSquared |
| FULL-07 | L | Fixed | wave 3 — fix 99221b6 |
| FULL-08 | L | Fixed | wave 1 — fix a0b961b |
| PM-03 | M | Fixed | wave 3 — fix 168ca832e7 · red check_docs_truth.py |
| PM-05 | L | Fixed | wave 4 — fix fa57fd66d2 |
| REL-01 | H | Fixed | wave 1 — fix 573985d · red client/test/coach_stream_timeout_test.dart |
| REL-02 | M | Fixed | wave 2 — fix 37b4507 · red TestOpen_StatementTimeoutCutsLongQueries |
| REL-03 | M | Fixed | wave 2 — fix 49a2bfa · red TestReapStaleAIRuns_PurgesRowsPastRetention |
| REL-04 | M | Fixed | wave 3 — fix fc803f9dd8 · red check_docs_truth.py |
| REL-05 | L | Fixed | wave 2 — fix 787c8ee |
| REL-07 | L | Fixed | wave 4 — fix 86c15a0 |
| SEC-02 | M | Fixed | wave 2 — fix 0e860ea51e · red TestFetchGuideFileStopsReadingAtTheCap |
| SEC-03 | M | Fixed | wave 2 — fix 6f6a9cc · red server/internal/api/single_replica_test.go |
| SEC-04 | M | Fixed | wave 4 — fix 4a7cb5b |
| SEC-06 | L | Fixed | wave 2 — fix 1a436569e7 · red TestParseRepoURLRejectsURLShapingCharacters |
| SEC-07 | L | Fixed | wave 2 — fix 6c81f73 · red TestUpdateMeAvatarURLSchemeDoor |
| SEC-08 | L | Fixed | wave 2 — fix 711dcd7 · red TestSharesWorkspaceWith_TrueForCoMembers |
| TEST-01 | H | Fixed | wave 4 — fix bcfe8b6 |
| TEST-02 | H | Fixed | wave 1 — fix d880eea |
| TEST-03 | M | Fixed | wave 4 — fix 8746ff5cd5 |
| TEST-04 | M | Fixed | wave 3 — fix c7849ac |
| TEST-05 | M | Fixed | wave 4 — fix d16be2fe2a |
| TEST-06 | M | Fixed | wave 3 — Fixed by the integrator across the migration boundary: circuit functions take a row id (20261011100000), rules pinned on an owned row |
| TEST-07 | L | Fixed | wave 3 — fix 6b11ba2 |
| TEST-08 | L | Fixed | wave 4 — fix e02c5c1 · red docs/review/2026-10-10-multirole-3/checks/product-status-loo… |
| USER-01 | H | Fixed | wave 1 — Fixed in wave 1 + integrator seam: the conversation GET now carries meta.pendingStage, so reload shows the proposal; confirm/decline live in the coach panel |
| USER-03 | H | Fixed | wave 2 — fix 8d0fd2f |
| USER-04 | M | Fixed | wave 4 — fix 58ec57b |
| USER-06 | L | Fixed | wave 3 — fix 6ecf8e1ded · red client/test/appearance_locale_seed_test.dart |
| USER-07 | L | Fixed | wave 3 — fix f62f98b27f · red client/test/coach_strapline_test.dart |
| USER-08 | L | Fixed | wave 2 — Fixed by the integrator across the ownership boundary: the stable '[interrupted]' token renders in the reader's locale |
| USER-09 | L | Fixed | wave 3 — fix ba6ad5e · red client/test/dashboard_capture_route_test.dart |
| PM-02 | M | Fixed | Deploy landed 2026-10-11 (`b6c34c4`): the behind-`main` qualifier is dropped, client/README states the deploy date, and the docs-truth check's deploy-skew arm retired with it |
| PM-04 | M | Fixed | Deploy landed 2026-10-11: `/api/v1/auth/config` reports `mailEnabled` live, so the client's unknown-state handling answers a real field |
| USER-05 | M | Fixed | `v0.1.0` tagged and published 2026-10-11 (four targets + checksums + bootstrap.sh); the repository went public the same day after a privacy sweep (no committed `.env`, no tokens or private keys in tree or history — the only hit is a code comment and the Supabase anon key, which is public-by-design), and the anonymous `curl \| bash` path is verified: bootstrap 200, tarball checksum OK, anonymous clone OK |
| COMP-03 | M | Satisfied by the report | Satisfied by the report itself: the score table carries the recalibrated 8.0 with the reason |
| COMP-05 | L | Satisfied by the report | Satisfied by the plan: bucket distribution stated; the verify contract adds mutation checks for IMPROVE rows |
| COMP-06 | L | Satisfied by the report | Satisfied by the report: 379/379 used throughout |
| CQ-04 | L | Parked | Parked by the brief's own pure-move gate: the delta-counter triplication is not a pure move (three providers' streaming bodies diverge); consolidation is a behavior-risk refactor beyond a Low finding — recorded here, not silently dropped |
| FEAT-03 | H | Fixed | Re-keyed and exercised live 2026-10-11: the owner's fresh (send-scoped) key synced to Railway + `.env` + both hosts' stores, redeployed; admin mail probe `status:"sent"`, and the full pass ran against the provider — invitation delivered (emailed code == inline code, redeemed by register 201), magic link read from the delivered email and consumed (tokens issued), login code read and verified, forgot/reset token redeemed and the new password logs in ([changelog](../../changelogs/2026-10-11-owner-deploy-and-mail-truth.md)) |
| PM-01 | H | Fixed | Reconciled 2026-10-11 with live evidence: Railway read (all mail vars present by name), the deployed `/auth/config` reports `mailEnabled:true`, and the admin mail probe answered `resend returned 401 Unauthorized` — every stored key copy (Railway, root `.env`, both hosts' stores) is revoked. The three disagreeing records now state this one truth; the re-key is the single remaining mail step (FEAT-03) |
| REL-06 | L | Fixed | The invitation email ran in production 2026-10-11: minted through the admin API, delivered by the provider (verified domain sender), the emailed link's code matched the inline code, and a fresh account registered through it — the mail path's first production delivery, click-through completed the same hour |
| SEC-01 | H | Fixed | Deployed 2026-10-11 and confirmed: `insideout migrate` against the hosted database reports already up to date — all seven migrations (four `20261010*` + three `20261011*`) applied |
| SEC-05 | L | Fixed | The live provider exercise ran 2026-10-11 and did its job: the admin mail probe reported the provider's 401 (the revoked key), and the LLM leg took a real hosted coach turn. The email-auth posture is now known — untrusted until the re-key, with a probe that tells the truth |
| USER-02 | H | Fixed (funnel rehearsed to the provider) | Deployed 2026-10-11; invitation minted through the admin API, register completed 201 with the invite code, and one real LLM coach turn ran on the hosted box (model answered, clean message_end). The mailed halves (magic link, login code, reset) reached the provider and exposed the revoked key — they complete with the Resend re-key ([changelog](../../changelogs/2026-10-11-owner-deploy-and-mail-truth.md)) |
