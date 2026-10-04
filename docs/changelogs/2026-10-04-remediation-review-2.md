# 2026-10-04 — Remediation of multirole review 2 (waves 1–3)

**Scope:** every actionable finding from
[review 2](../review/2026-10-04-multirole-2/README.md) — the 7 confirmed
Highs, the wave-1 one-liners, and the wave-3 hardening decisions. Plan:
[`docs/plans/2026-10-04-remediation-review-2/`](../plans/2026-10-04-remediation-review-2/2026-10-04-remediation-review-2.md).
Wave 4 (feature slices) stays parked by design.

Commits: `e96b774` (wave 1), `5dc986a` (AIDEV-09), `8dfbc11` (USER-09 +
USER-10), `5767a0e` (USER-11), `c5d4b16` (FEAT-03 + setState sweep),
`0eb57e9` (REL-13), `ac0cf22` (CC-01 + CC-03), `073def0` (CQ-01).

## What changed

- **AIDEV-09 (High)** — the fresh-context critic now sends a real user
  message (the PRD rides in the system prompt; the turn directs the forced
  tool call), and `AnthropicStreamer.StreamChatForcingTool` rejects an
  empty payload locally with a named cause instead of a provider 400.
  Failing test first: a recording stub asserted the payload.
- **USER-09 (High)** — Drop visibly drops: `ListIdeas` excludes
  `status='dropped'` (DB-gated test first, run against the real
  PostgreSQL), a confirm dialog uses the previously-dead `idea.dropConfirm`
  key, and two widget tests pin cancel/confirm. A pre-existing
  `setState(() => _future = …)` assertion bug surfaced and was fixed.
- **USER-10 (High)** — the web coach SSE no longer bypasses 401 recovery:
  `fetchSseAttempt` opens eagerly (status known before frames), and
  `coachSseWithRefresh` applies the interceptor's exact policy — one
  single-flight refresh, one retry with the rotated token, else
  `clearTokens` so the router lands on the auth door. Five policy tests.
- **USER-11 (High)** — one l10n error mapper (`errors_l10n.dart`): status →
  key, backoff codes → throttled, transport → network, generic fallback;
  routed through 12 data-page error branches with 9 new keys in en+zh.
  Auth forms are deliberately excluded (a login 401 is "wrong credentials",
  not "session expired").
- **FEAT-03 (High)** — agent proposals get their web decision loop:
  Accept/Reject call the existing `/agent/proposals/{uid}/decision`
  endpoint (accept applies structured items, matching CLI/MCP), and the
  generic edit/delete menu is hidden for that kind. Widget tests cover
  both decisions. Same commit fixed the `setState`-returns-Future class
  in six pages.
- **REL-13/SEC-10 (High)** — per-IP throttle no longer collapses to the
  shared proxy IP: Railway's edge appends the real client IP to
  X-Forwarded-For (documented behavior; spoofed values stay left), and
  nginx now passes the chain through instead of appending its hop, so the
  server's rightmost non-empty entry is the per-client IP. Unit tests pin
  the Railway chain and trailing-empty robustness; topology documented.
- **Wave 1 one-liners** — `window.open`'s null return captured (popup
  fallback live, CQ-02 backstop), CI pinned to the Dockerfile Flutter
  3.44.0, skip-guard widened to the whole suite, guarded `isoDate` helper
  adopted at three call sites, revoke delay named, PRODUCT.md names the
  running v1 authority model.
- **CC-01** — `deployment` webhooks map into the evidence pipeline with a
  release kind (environment + sha); `deployment_status` refinement is a
  later slice.
- **CC-03** — backup story: `scripts/backup-db.sh` (owner-DSN dump — the
  app role is RLS-subject and pg_dump fails against it by design) plus
  [backup-restore.md](../usage/backup-restore.md); the managed half
  (Supabase automated backups) is called out as the operator's toggle.
- **CQ-01** — shared `RoadmapBandScaffold` extracted for the Bands and
  Progress views.

## Verification

- Server: `go build ./...`, `go vet ./...` clean; full
  `go test ./... -count=1` green with 0 skips (store suite 834s against
  the shared deployment database; api + agent packages re-run after the
  last server-side change).
- Client: `flutter analyze --no-fatal-infos` 0 warnings/errors (20 infos =
  accepted baseline); `flutter test` 77/77 (was 65: +2 ideas, +5 SSE
  policy, +3 error mapper, +2 project page).
- New failing tests written first for: critic payload, drop visibility
  (DB-gated), SSE refresh policy. The XFF robustness loop and guard were
  verified by their new tests in the same change.
- Backup runbook walked on two scratch postgres:18 containers: dump →
  restore → `migrate` no-op → 23 tables present.
- **Not run, honestly:** a live provider exercise of the fixed critic
  (local env carries no `INSIDEOUT_LLM_API_KEY` — operator-gated, same
  tier as the signed-in page load; the deployed critic's next
  critique-stage conversation is the live proof) and a two-source-IP
  per-client proof of the throttle (impossible from one machine; the
  deployed check is the domain login path staying green).

## Deploy

Both services were deployed after `073def0` (server `railway up` build +
deploy SUCCESS; `app` image rebuilt with the nginx pass-through config),
and the domain answered 200 on the landing page / 401 on `/api/v1/me`
without a token.

## Operator actions (unchanged, human-only)

- Rotate the Resend mail key if local transcripts can leave the machine.
- The signed-in hosted page load (PM-02 release gate).
- Confirm Supabase automated backups are enabled (CC-03's managed half).
