# 2026-10-04 — Remediation of multirole review 2 (waves 1–3)

**Status:** Finished 2026-10-04. **Source:**
[`docs/review/2026-10-04-multirole-2/40-remediation-plan.md`](../../review/2026-10-04-multirole-2/40-remediation-plan.md)
(register IDs below refer to it). Durable state:
[`2026-10-04-remediation-review-2.track.yaml`](2026-10-04-remediation-review-2.track.yaml).
Closure record:
[`docs/changelogs/2026-10-04-remediation-review-2.md`](../../changelogs/2026-10-04-remediation-review-2.md).

**Goal (observable):** every actionable finding from review 2 is fixed — the 7
confirmed Highs, the wave-1 one-liners, and the wave-3 hardening decisions —
with the full suite green (`go test ./... -count=1` 0-skip, `flutter test`,
env batteries), both services deployed, and the domain verified.

**Not in this goal (parked):** wave 4 feature slices (Decision Log UI, Branch
slice 1, per-project authority) — roadmap work, each needs its own plan;
Flutter toolchain upgrade beyond the one-line CI pin; the two human-only
actions (Resend key rotation, signed-in hosted page load).

## Wave 1 — one-liners, one commit

- [x] FS-01 `download_web.dart` capture `window.open` null → `return false`
      (fallback live); CQ-02 feature_wiring substring on `export_page.dart`.
- [x] ARCH-01 `ci.yml` pin `flutter-version: 3.44.0` (match Dockerfile).
- [x] TEST-09 widen skip-guard grep to the single verbose `go test ./...` run;
      drop the duplicate store-only re-run.
- [x] CQ-03/CQ-04 guarded date helper from `chrome/time_ago.dart` adopted at
      `roadmap_node_card.dart:53,164` + `roadmap_progress_view.dart:104`.
- [x] CQ-05 name `_revokeDelayMs` in `download_web.dart`.
- [x] CC-02 PRODUCT.md status sentence: admin-approves-PRDs is the running v1
      authority model.

Acceptance: `go test ./... -count=1` (0 skips), `flutter test`,
`flutter analyze --no-fatal-infos`, CI green on push.

## Wave 2 — confirmed Highs (failing test first where behavior risk warrants)

- [x] AIDEV-09 critic one-message payload + wire-level test asserting
      forced-tool requests never marshal empty `messages`; live critic
      exercise per streaming practice — **live provider exercise is
      operator-gated** (local env has no `INSIDEOUT_LLM_API_KEY`; the key
      lives only in Railway), same tier as the signed-in page load.
- [x] USER-09 drop visibility: DB-gated test first (`ListIdeas` excludes
      `status='dropped'` for the author), confirm dialog on `idea.dropConfirm`,
      localize the dropped state.
- [x] USER-10 web SSE 401: single-flight refresh, retry once with rotated
      token, else `clearTokens` → auth door.
- [x] USER-11 l10n error mapper (`ApiException` → key, generic fallback),
      routed through all ~12 error branches; zh + en strings. (Auth forms
      deliberately excluded: a login 401 means "wrong credentials", not
      "session expired" — their copy is its own slice.)
- [x] FEAT-03 web proposal loop: `kind == 'agent_proposal'` gets
      Accept/Reject via the existing decision endpoint; edit/delete hidden.

Acceptance: same suites + widget tests for the client rows; live SSE exercise
against a local server.

## Wave 3 — hardening decisions

- [x] REL-13/SEC-10 XFF: Railway's edge appends the real client IP to XFF
      (documented; client-supplied spoofs stay left); nginx now passes the
      chain through instead of appending its hop; rightmost non-empty entry
      wins (trailing-empty robustness + Railway-chain unit tests); topology
      documented in docs/usage/deployment.md. Live per-client proof needs
      two source IPs — the deployed check is the domain login path staying
      green.
- [x] CQ-01 shared band scaffold for `_BandRow` + `_ProgressBand`
      (`roadmap_band_scaffold.dart`).
- [x] CC-03 backup story: `scripts/backup-db.sh` (owner-DSN dump — the app
      role is an RLS subject and pg_dump fails against it by design) +
      docs/usage/backup-restore.md; walked end-to-end on two scratch
      postgres:18 containers (dump → restore → migrate no-op → 23 tables).
      The managed half (Supabase automated backups) is an operator toggle,
      named in the doc.
- [x] CC-01 deployment events: `deployment` webhooks map into the evidence
      pipeline with the release kind (environment + sha). deployment_status
      (success/failure refinement) is a later slice, noted in code.

Acceptance: two-hop throttle test green; scaffold refactor keeps widget tests
green; restore doc walked; webhook decision recorded.

## Ship loop

- [x] Explicit-staging commits, push to main, CI green (run 37205150447:
      server + client + env jobs all success on `073def0`).
- [x] `railway up --service server` + `app` (both SUCCESS 2026-10-04
      21:23 +08); domain checks: landing 200, SPA healthz 200,
      `/api/v1/me` 401 unauthenticated, three bad-login probes all 401
      through the new XFF pass-through chain.
- [x] Records: dated changelog, board rows closed, HANDOFF updated.
