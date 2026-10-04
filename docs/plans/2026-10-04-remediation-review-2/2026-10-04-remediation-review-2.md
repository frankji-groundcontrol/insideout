# 2026-10-04 — Remediation of multirole review 2 (waves 1–3)

**Status:** In flight 2026-10-04. **Source:**
[`docs/review/2026-10-04-multirole-2/40-remediation-plan.md`](../../review/2026-10-04-multirole-2/40-remediation-plan.md)
(register IDs below refer to it). Durable state:
[`2026-10-04-remediation-review-2.track.yaml`](2026-10-04-remediation-review-2.track.yaml).

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

- [ ] AIDEV-09 critic one-message payload + wire-level test asserting
      forced-tool requests never marshal empty `messages`; live critic
      exercise per streaming practice.
- [ ] USER-09 drop visibility: DB-gated test first (`ListIdeas` excludes
      `status='dropped'` for the author), confirm dialog on `idea.dropConfirm`,
      localize the dropped state.
- [ ] USER-10 web SSE 401: single-flight refresh, retry once with rotated
      token, else `clearTokens` → auth door.
- [ ] USER-11 l10n error mapper (`ApiException` → key, generic fallback),
      routed through all ~12 error branches; zh + en strings.
- [ ] FEAT-03 web proposal loop: `kind == 'agent_proposal'` gets
      Accept/Reject via the existing decision endpoint; edit/delete hidden.

Acceptance: same suites + widget tests for the client rows; live SSE exercise
against a local server.

## Wave 3 — hardening decisions

- [ ] REL-13/SEC-10 XFF: verify the real chain (Railway edge + nginx append),
      fix the double-append (nginx passes the edge's XFF through / hop depth
      explicit), two-hop unit test, document the topology.
- [ ] CQ-01 shared band scaffold for `_BandRow` + `_ProgressBand`.
- [ ] CC-03 backup story: confirm whether Railway Postgres backups are on;
      otherwise nightly `pg_dump` script + restore doc walked once on a
      scratch database.
- [ ] CC-01 deployment events: map `deployment` webhooks into evidence with a
      release kind, or remove the promise from guide template + PRODUCT.md.

Acceptance: two-hop throttle test green; scaffold refactor keeps widget tests
green; restore doc walked; webhook decision recorded.

## Ship loop

- [ ] Explicit-staging commits, push to main, CI green.
- [ ] `railway up --service server` + restart `app`; domain checks
      (landing 200, API 401 unauthenticated, live coach SSE through nginx).
- [ ] Records: dated changelog, board rows closed, HANDOFF updated, review
      register statuses noted where rows are now fixed.
