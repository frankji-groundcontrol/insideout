# 2026-08-23 — Canvas presence hardening: reconnect + live multi-session proof

Status: **in flight**.

## Why this exists

The board row for [restore-ink-seal](2026-08-19-restore-ink-seal.md) still
names "real-time multi-user presence on the canvas" as the open follow-on,
but the [frontier](../changelogs/2026-08-22-frontier.md) and
[depth](../changelogs/2026-08-22-depth-thread.md) threads already shipped it:
the Flutter roadmap page registers a per-tab session on the presence SSE
stream, renders who-is-here chips, sends throttled cursor moves, and paints
other sessions' labeled cursors. What is actually open is hardening and
proof, not the feature itself:

1. **No reconnect.** `_watchPresence` opens the stream once with
   `.catchError((_) {})`. Any drop (server deploy, proxy blip, laptop sleep)
   silently kills presence and cursors for that tab forever. Railway server
   deploys drop every stream, so presence dies exactly when the app updates.
2. **No UI-level multi-session proof.** The frontier verified cursor events
   at stream level; two real tabs showing each other chips + cursors has not
   been captured.

## Scope

- `_RoadmapPageState._watchPresence` becomes a retry loop: on stream end or
   error, re-open after a capped backoff (0.5 s doubling to 5 s) while the
   page is mounted. Fatal auth/project errors (401/403/404) stop the loop
   instead of hammering the API.
- Widget test with a fake `ApiClient.presenceStream` that delivers one
   snapshot, dies, and succeeds on the next call: chips must come back
   (reconnect observed).
- Live two-tab verification against a local build served over the real API:
   tab B's chip and cursor appear in tab A and vice versa.
- Board + plan hygiene: this plan owns the follow-on; the restore-ink-seal
   row and HANDOFF point here once closed.

## Out of scope

- Cursor-dot idle fade (polish; dots already prune when a session leaves).
- Presence on other pages (roadmap is the collaborative canvas).
- Server changes (registry, TTL, routes stay as verified 2026-08-22).

## Checklist

- [x] Reconnect loop with capped backoff + fatal-status stop
- [x] Reconnect widget test
- [x] Full client suite + analyze green
- [x] Live two-tab chip/cursor capture (local build, real API) — plus a
      web SSE transport fix the capture exposed (dio XHR never streams)
- [ ] Deploy bundled with the font fix (`railway up --service app`),
      then verify hosted and purge scratch data
