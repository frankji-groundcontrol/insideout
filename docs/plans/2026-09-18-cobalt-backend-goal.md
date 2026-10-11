# Goal prompt — Cobalt Flutter against the live API

Paste the block below into a new agent (or Cursor Goal). It is
self-contained. Do not restyle. Do not invent APIs.

---

Make the **already-restyled Cobalt Flutter client** work against the
**real Go backend**. Sign in. Drive the product loop on real rows.
Fix only the wiring that breaks. Do not explore, vote, restyle, or
redesign.

## Success

A signed-in user on Flutter web, talking to a real `/api/v1` (hosted
or a local `scripts/dev.sh` server — not a fake `ApiClient`), completes
Idea → version → human Commit → timed Roadmap → Git evidence on the
Cobalt surfaces. Dock hops use real workspace / PRD / project IDs.
Coach SSE streams live if the API is up. Light and dark, zh-CN and
en-US still work. No new routes, schema, or product verbs.

## Why this is open

PR [#3](https://github.com/frankji-groundcontrol/insideout/pull/3)
(`cursor/vibrant-ink-seal-198b`) restyled pixels. The browser walk
used a **local preview with a fake API**. `GET
https://insideout.yalotein.net/api/v1/me` is alive (`401` when signed
out). This environment had no `.env` and hosted signup is
invitation-only, so no real session was opened.

The live entrypoint is already real
([`client/lib/main.dart`](../../client/lib/main.dart) +
[`Session`](../../client/lib/session/session.dart) +
[`ApiClient`](../../client/lib/api/client.dart)). This slice is
**connect, login, exercise, fix breakages** — not a second theme.

## Source of truth (in order)

1. This prompt and
   [`docs/plans/2026-09-18-cobalt-backend.md`](2026-09-18-cobalt-backend.md).
2. Restyle record:
   [`docs/plans/2026-09-18-cobalt-flutter.md`](2026-09-18-cobalt-flutter.md),
   [`docs/changelogs/2026-09-18-cobalt-flutter-restyle.md`](../changelogs/2026-09-18-cobalt-flutter-restyle.md).
3. Transport:
   [`docs/architecture/frontend.md`](../architecture/frontend.md),
   [`docs/usage/local-development.md`](../usage/local-development.md),
   [`docs/HANDOFF.md`](../HANDOFF.md).
4. SSE practice:
   [`docs/practices/2026-07-21-live-exercise-streaming-endpoints.md`](../practices/2026-07-21-live-exercise-streaming-endpoints.md).
5. Lock screens stay visual-only:
   [`docs/design-system/cobalt/`](../design-system/cobalt/README.md).

## How to reach a real API

Prefer the hosted API (CORS already allows localhost):

```bash
cd client && flutter run -d chrome --web-port=5173 --web-hostname=localhost \
  --dart-define=API_BASE=https://insideout.yalotein.net/api/v1
```

If `DATABASE_URL` is already set, a local server is also fine:

```bash
./scripts/dev.sh -C server go run ./cmd/insideout migrate
./scripts/dev.sh -C server go run ./cmd/insideout seed   # only if you need the local demo row
./scripts/dev.sh -C server go run ./cmd/insideout
./scripts/dev.sh -C client flutter run -d chrome \
  --dart-define=API_BASE=http://127.0.0.1:8080/api/v1
```

Do **not** `env.sh init` unless the user asked to set up this machine.
Never read or print `.env` / `.env.local`. Never invent a hosted user.
Signup is invitation-only
([plan](2026-08-25-invitation-only-signup.md)). If you cannot log in,
**stop and ask** for a login path (existing account, magic link, or
operator). The local seed identity lives in
[`server/cmd/insideout/seed.go`](../../server/cmd/insideout/seed.go) —
use it only against a DB you just seeded; do not write it into
changelogs or the PR.

Hosted production still serves the **pre-restyle** bundle until someone
runs `railway up --service app` after this branch merges. Verify with
`flutter run` against the hosted API first. Do not redeploy unless the
user asked.

## Wiring to keep (already in the tree)

| Piece | File | Must stay true |
| --- | --- | --- |
| Tokens | `Session` + `flutter_secure_storage` | `accessToken` / `refreshToken` from login JSON; cookies unused |
| Paths | [`router.dart`](../../client/lib/router.dart) | `/dashboard`, `/workspace/:id/ideas`, `/prd/:id`, `/prd/:id/versions`, `/projects/:id`, `/projects/:id/roadmap` |
| Dock IDs | [`loop_focus.dart`](../../client/lib/chrome/loop_focus.dart) + [`product_dock.dart`](../../client/lib/chrome/product_dock.dart) | Remember workspace / PRD / project from real loads; silent setters in `initState` |
| Coach | [`coach_panel.dart`](../../client/lib/features/prd/coach_panel.dart) | `sendCoach` + `CoachHandlers` against live SSE |
| Capture | [`ideas_page.dart`](../../client/lib/features/workspace/ideas_page.dart) | `createIdea` then reload |
| Commit | [`versions_page.dart`](../../client/lib/features/prd/versions_page.dart) | `prdReadiness` + `commitPrd` |
| Roadmap | [`roadmap_page.dart`](../../client/lib/features/project/roadmap_page.dart) | `createRoadmapNode` lands in a Now/Next/Later band |
| Evidence | [`project_page.dart`](../../client/lib/features/project/project_page.dart) | `getProject` / repo / updates |

Web path strategy stays `usePathUrlStrategy()` in `main.dart` (the
hash URLs in the restyle screenshots were the discarded preview).

## Likely breakages to hunt (fix only if real)

- Dock goes to `/dashboard` because `LoopFocus` never saw a real ID
  (open Ideas / PRD / project, then dock again).
- `remember*` during `build` vs `initState` (keep setters silent).
- Capture / Commit / add-Now / Coach send fail (status, body, auth).
- Coach SSE buffered or dead
  ([practice](../practices/2026-07-21-live-exercise-streaming-endpoints.md)).
- Empty / error states after a real 401 / 404 / 5xx.
- CORS only if `flutter run` is not `localhost` / `127.0.0.1`.

Do not add a preview `ApiClient`. Do not ship `tool/lock_preview.dart`.

## Forbidden

- Restyling, new hues, new layouts, reopening Cobalt vs Ink & Seal.
- New backend routes, schema, or writes outside `insideout`.
- Mocks. Fake `ApiClient` for the “done” walk.
- Reading or printing `.env` / `.env.local`, tokens, or DSNs.
- Registering on hosted (invitation-only). Inventing users.
- Inventing product behavior. The loop stays
  Idea → version → human Commit → timed Roadmap → Git evidence.
- English-only chrome copy — keep `l10n` (zh-CN default, en-US).

## How to work

1. Open a dated plan (or drive
   [`2026-09-18-cobalt-backend.md`](2026-09-18-cobalt-backend.md))
   **before** editing. Keep `docs/plans/README.md` and
   `docs/HANDOFF.md` current.
2. Get a real session first. Then the smallest wiring fix that unblocks
   the next loop step. TDD when you change client request/response
   code: failing test against the real contract, then the fix.
3. Stay on `cursor/vibrant-ink-seal-198b` (PR #3) if it is still open.
   If it has merged, branch from `main` with the repo’s `cursor/…`
   prefix. Do not start a parallel visual exploration.
4. Files under ~350 lines. If a file cannot be split now, file a dated
   modularity issue.
5. Changelog the wiring (not another theme rewrite). Record frontend
   comments in `docs/design-qa/`.

## Verification (required before done)

```bash
(cd client && flutter analyze --no-fatal-infos && flutter test)
```

Then a **real** signed-in browser walk against the real `API_BASE` (not
a fake client, not a single screenshot):

- log in (and hydrate after a refresh)
- light and dark, zh-CN and en-US
- Home lists **server** workspaces
- Ideas: capture one idea; it remains after reload
- PRD + Coach: send a message; deltas arrive live
- Commit: form a version; it appears in the stack
- Roadmap: add a Now item; it appears in a band
- Evidence: existing Git / updates load
- dock hops those surfaces without falling back to `/dashboard`
- empty / error states you hit

If you touch store code (you should not), run the `DATABASE_URL`-gated
authz tests. Never print secrets.

Repo contract: [`AGENTS.md`](../../AGENTS.md),
[`docs/index.md`](../index.md).

*Superseded 2026-10-11: "Hosted production still serves the pre-restyle
bundle until someone runs `railway up --service app`" no longer holds —
both services were deployed and domain-verified 2026-10-04, and the owner
deploy 2026-10-11 shipped `b6c34c4` (server) / `6b1f24f` (app) with
`v0.1.0` published
([owner-gap plan](2026-10-11-owner-gap-closure.md)).*
