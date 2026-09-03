# BUG-015 — SetSignupMode sent POST to a PUT-only route

**Status: fixed 2026-09-03** (same-day discovery and fix).

## Symptom

`insideout signup-mode <mode>` failed against the real API with
`405 Method Not Allowed` — the admin could read the registration policy
but never switch it from the CLI. Found by the
[2026-09-03 surface-parity](../changelogs/2026-09-03-surface-parity-gaps.md)
live matrix, not by tests.

## Root cause

The invitation-only slice (2026-08-25) registered the route as
`PUT /api/v1/admin/settings` (`server/internal/api/admin.go:23`) while
apiclient `SetSignupMode` (`server/internal/apiclient/auth.go`) used
`postRaw`. The slice's live verification covered the web admin panel
(which sends PUT) and the read path (`GET /auth/config` is public), so
the CLI write path was never exercised end-to-end. httptest unit tests
pinned the CLI's own behavior — including the wrong method — because
they assert against what the client sends, not against what the server
accepts. Method disagreement is invisible unless a test (or a live run)
puts the two sides together.

## Fix

`SetSignupMode` now sends `PUT`, pinned by
`TestSetSignupModeUsesPut` (method + path + body), then verified live:
`signup-mode open` round-trip on a scratch server.

## Lesson

For client projections of an API (apiclient → CLI → MCP), a
method/path pinning test proves the client is *consistent*, not
*correct*. Every new projected verb needs one live call against a real
server before the slice closes — the same rule the repo already keeps
for SSE endpoints, applied here to plain JSON verbs.
