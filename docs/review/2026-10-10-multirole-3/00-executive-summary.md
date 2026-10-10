# 00 · Executive summary

Review 3 over `main` @ `754c502`. 83 findings (0 Critical, 17 High, 33 Medium,
33 Low); all 16 reviewer Highs CONFIRMED by skeptics. Full detail in the
[register](01-findings-register.md); execution shape in the
[remediation plan](40-remediation-plan.md).

## The top ten, ordered by what they cost the product

1. **ARCH-01 — the server image cannot build.** `server/go.mod` demands Go
   1.26.0; `server/Dockerfile` pins `golang:1.25-alpine`. Every documented
   image path (`docker compose up -d server client`, the Railway server
   service) dies at `go mod download` — the exact BUG-005 symptom — and no CI
   lane builds the server image. The coordinator's reproduction **closed the
   fix decision**: `golang.org/x/sys` and `golang.org/x/term` already require
   1.26.0, so the base image moves to `golang:1.26-alpine`; lowering go.mod is
   off the table. Fix size: 1 line + one CI lane.
2. **AIDEV-01 — the finalize gate reads model-rewritable state.** Any
   critique-stage coach reply that parses as the ranked list wholesale-replaces
   the stored `critic_findings` (new IDs, statuses reset), so a restatement
   that omits an open blocking finding satisfies the human-override gate the
   product's trust story depends on. Merge-by-content instead of replace. ~30
   lines + test.
3. **FULL-01 — the invitation email lies 60x about its own link.** The HTML
   body renders a 7-day invite as "expires in 168 minutes" (hours passed into
   a template hardcoding "minutes"). This is the first message every external
   user receives, on a funnel that has never run live. One line + copy-pinning
   tests (FULL-08, TEST-02).
4. **USER-01 — the stage-confirm door has no handle.** `POST
   /conversations/{id}/stage` exists and is the only way the persisted stage
   advances — and nothing calls it: not the web client, not the CLI, not MCP.
   The UI optimistically shows the proposed stage and reverts on refresh, and
   the completion gate never sees a human-ratified stage. One client control +
   honest "proposed" labeling.
5. **REL-01 — native coach turns die at 30 silent seconds.** The shared Dio
   client's `receiveTimeout: 30s` kills the native SSE stream while the server
   tolerates 90s of provider silence and the client's own idle timer waits
   75s. Large first calls and the auto-tighten retry are the failure window. A
   per-request `receiveTimeout: Duration.zero` override.
6. **FEAT-02 — idea conversion throws away the idea.** ConvertIdea carries
   only the title into the PRD and conversation; the body the user wrote never
   reaches the coach, against PRODUCT.md's "preserves the original title and
   body as coaching context". Insert the body as the first conversation
   message inside the transaction.
7. **SEC-01 + TEST-01 — the security fixes are written but not shipped, and
   the upgrade path is untested.** Four 2026-10-10 migrations (removed-member
   RLS closure, short invite-code rotation) are merged but unapplied on the
   hosted database, and every CI path migrates a fresh empty DB — the
   migrations' first meeting with real rows will be production. Deploy is the
   owner step; the CI upgrade-replay lane is ours.
8. **FEAT-01 — the coach speaks one of two promised languages.** Every LLM
   prompt (coach stages, critic, planner) is hard-coded Chinese; PRODUCT.md
   makes English and Simplified Chinese first-class. Thread the client's
   existing locale into conversation creation and select prompts from it. The
   largest bounded slice in the plan.
9. **USER-03 — the offline coach is a disguised stub.** Without an LLM key the
   template coach replies with canned Chinese text instructing the reader to
   edit server env vars, and nothing in the stream or UI distinguishes it from
   real coaching. Expose the streamer identity in-band and badge it.
10. **TEST-02 — the only entrance flows have no positive test.** Invite,
    magic-link, login-code, and reset flows are tested for validation 400s
    only; no test anywhere mints a link and consumes it. The dev-mode mail body
    log is the seam; ~120 lines of request→consume round trips.

## Effort by bucket

| Bucket | Rows | Shape of the work |
|---|---|---|
| FIX (22) | 11 High | Wave 1 of the plan: gate integrity, planner fence, email copy, Dockerfile toolchain, SSE timeout, idea body, stage-confirm surface — all red-test-first, file-disjoint |
| HARDEN (22) | 5 High | Wave 2: provider taxonomy + output caps, turn bounds, GitHub fetch/URL hardening, store statement/reaper bounds, in-memory registry caveat, mounted guards, offline disclosure, URL scheme checks |
| IMPROVE (25) | 1 High (COMP-01) | Wave 3: docs-truth sweep (stale counts, layout tree, board row, README claims), prompts contract rewrite, parser validation, client polish, dead code |
| ADD (8) | 2 High | Wave 4: locale threading, migration-upgrade lane, hosted smoke mode, validation-evidence slice, lead-14 promise register, CONTRIBUTING/SECURITY, product-gap issue consolidation |

Owner-blocked (deploy, mail/LLM keys, first release tag): PM-01, PM-02,
PM-04, SEC-01, USER-02, FEAT-03, REL-06, SEC-05, USER-05 — listed in the
plan's [owner list](40-remediation-plan.md#owner-blocked-not-workflow-work)
and already in [HANDOFF](../../HANDOFF.md).

## The convergent recommendation

Three roles independently reached the same structural conclusion: the next
milestone should be a **deploy milestone, not a build milestone**. The tree is
verified to a degree most repos never reach and entirely unshipped — no `v*`
tag, hosted deployment 90 commits stale, both provider integrations (LLM
coach, mail funnel) never exercised live. Fix the toolchain floor (ARCH-01),
land wave 1, then deploy and tag: that single owner sequence un-darks the
whole register's FIX bucket and converts the mail/LLM "never run live" rows
from unknown to verified.
