## Reliability & ops review — frontend

**Scope.** `client/lib/api` (client.dart, sse.dart, sse_transport_web.dart,
errors.dart, errors_l10n.dart), `client/lib/session` (token_store.dart,
session.dart, boot.dart), `features/prd/coach_panel.dart`,
`features/project/roadmap_page.dart`, and the error surfaces on
dashboard_page.dart and versions_page.dart. The question this pass asks: if
the network misbehaves, does the client degrade predictably — and the
honest answer is "sometimes, and the worst cases are the silent ones."

### What is genuinely solid

The auth-recovery core is the strongest part of the client. The refresh
interceptor in `client/lib/api/client.dart:47-71` correctly handles the
subtle parts: `/auth/*` 401s are terminal and untouched, the retry marker
in `options.extra` prevents loops, a failed refresh clears the session so
the router lands on the auth door, and `_refreshSingleFlight` (lines 77-83)
collapses a thundering herd of simultaneous 401s into one refresh. This is
not just plausible on paper — `client/test/api_token_refresh_test.dart`
exercises it against a real loopback HTTP server, including the
concurrent-requests case asserting exactly one refresh POST. The web-side
mirror in `coachSseWithRefresh` (sse.dart:167-190) re-applies the same
policy to the raw fetch path that bypasses dio, including clearing tokens
after a failed refresh so the user is not trapped in a "stream failed"
loop.

The SSE layer is likewise careful: `decodeSseBytes` (sse.dart:92-94)
copies each chunk before running the UTF-8 transformer, which is the
correct treatment for multibyte characters split across ReadableStream
chunks, and `parseSseBuffer` leaves partial frames in the carry buffer.
The server's writers (e.g. `server/internal/api/presence_routes.go:103`,
`roadmap_progress.go:64`) all emit LF-terminated `event:/data:` frames, so
the splitter matches the wire format in practice. The boot gate
(`main.dart:27-34`) completes on error as well as success, and the
roadmap retry loops are properly bounded — 500ms doubling to a 5s cap —
with 401/403/404 treated as fatal so a dead session does not spin.

### The big hole: there is no timeout anywhere

`BaseOptions` in client.dart:30-34 sets baseUrl, a content-type header,
and `validateStatus` — and no `connectTimeout`, `receiveTimeout`, or
`sendTimeout`. A grep across `client/lib` confirms the only
`Duration(seconds:)` timeouts in the tree belong to web-font loading and
UI countdown timers. This turns "the network is weird" from an error into
a disappearance, in three places:

1. **The boot gate can hang forever.** `Session.hydrate()` calls `_me()`
   and, on failure, `_refresh()` — both unbounded. `main.dart`'s
   `Future.wait(...).then(..., onError: boot.complete)` handles *errors*,
   but a black-holed connection (captive portal, firewall DROP rule) is
   neither resolution nor rejection. The app sits on `BootLoadingPage`
   indefinitely; the only escape is a reload on a better network.
2. **The coach composer can lock forever.** If the server accepts the
   coach POST and the upstream hangs mid-stream, no frames arrive, the
   fetch never settles, `sending` stays true, and the text field stays
   disabled (`coach_panel.dart:212`) with no cancel control.
3. **Every data page spinner can spin forever.** All the FutureBuilder
   pages await dio futures with no ceiling.

Tellingly, `errors_l10n.dart:38` has a `TimeoutException` branch and
`errors_l10n_test.dart:33` asserts it — a tested contract the runtime
cannot currently produce. The first limit to set before the next run is a
10s connect / 30s receive timeout on Dio, plus a ~15s timeout around
hydrate's network steps so the gate always lifts into a defined state.

### hydrate() conflates "offline" with "logged out"

`session.dart:52-60` nests `_me()` → `_refresh()` → re-`_me()`, and the
outer `catch (_) { user = null; await clear(); }` deletes both tokens for
*any* exception — a 401, but also a `SocketException` on a commuter train.
The result: an offline app start silently wipes the session, and the user
re-authenticates (magic-link email and all) for what was a dead tunnel,
not a dead session. The fix is a one-line distinction: clear storage only
for ApiException with status 401/403; treat transport errors as
"signed out for this launch" without touching the store.

### Error surfaces: three ways to be wrong

The dashboard and versions pages render `'${snap.error}'` directly
(dashboard_page.dart:98, versions_page.dart:73) — raw English exception
text as the entire page body in a zh-CN-first product, with no retry
button, while roadmap_page routes through `localizedError` correctly. The
coach panel has the opposite, subtler bug: it stores `e.message` (a
String), then hands that string to `localizedError` (coach_panel.dart:194),
which matches no typed branch and *always* returns `error.generic` — the
real failure classification is discarded before it reaches the UI. Worse,
the countdown label at line 199 is chosen by sniffing the English
substring `'unavailable'`, and the server's permit-queue 503 says "AI
service is busy, try again shortly" (`server/internal/agent/coach.go:150`)
— so a circuit event is labeled "rate limited". Store the exception, not
its message; branch on `code`/`status`, never on text.

One more classification wrinkle: `validateStatus: s < 500` means ordinary
HTTP 5xx arrives as a `DioException`, which the mapper reports as
"network" — so during a server deploy every page blames the user's
connection, and the `error.server` branch is unreachable for plain HTTP
calls. Checking `DioException.response?.statusCode` first fixes it.

### Lifecycle leaks on the roadmap page

Two findings compound here. First, the single `_retryTimer` field
(roadmap_page.dart:40) backs both backoff loops: `_backoff` assigns it
(line 45), so the second loop's timer clobbers the first loop's handle,
and `dispose` cancels at most one of two pending timers. The mounted
checks make this mostly harmless today, but it is exactly the kind of
shared-mutable-handle bug that bites the next edit. Second, and more
materially, the live streams are never actually cancelled: the web reader
loop (`sse_transport_web.dart:35-44`) has no `reader.cancel()` path, the
dio side has no CancelToken (the client.dart:505-508 comment admits the
page "closes by letting it die with dispose"), and the progress/presence
loops only re-check `mounted` after the current stream returns — which,
for a healthy connection, is never. Every roadmap visit parks suspended
loops and open connections until the server or the process dies.

The coach panel has the matching lifecycle bug in callback form:
`onDelta` (coach_panel.dart:80) and both catch blocks (lines 96, 99) call
`setState` with no `mounted` guard, so navigating away mid-stream throws
setState-after-dispose. The sibling callbacks got the guard; these three
did not.

### Token storage and the pivot

On web — the primary documented target — `SecureTokenStore` is just
flutter_secure_storage, whose web backend is localStorage with no secure
enclave and no encryption: any same-origin script can read both tokens.
Refresh-token rotation bounds the exposure window but does not remove it;
this is a documented-the-web-trade-off or move-to-HttpOnly-cookie
decision, not a code bug, but it should be written down in the
architecture docs rather than left implicit.

None of this blocks the neo-brutalist pivot, but one reviewed line does
intersect it: `brightnessIsDark` is computed by comparing the resolved
surface color to `Cobalt.darkGround` (coach_panel.dart:230-232) rather
than asking the theme. A four-step near-black ladder with more than one
dark ground hex will silently flip coach-panel polarity. Ask
`Theme.of(context).brightness` while you are already touching the file for
the timeout work.

### Priority order

1. REL-01: dio timeouts + hydrate timeout (lifts the boot-gate hang).
2. REL-02: stop clearing tokens on transport errors.
3. REL-03: coach-stream inactivity timeout + cancel.
4. REL-04/REL-08: exception-object error state; localizedError everywhere.
5. REL-05/REL-06/REL-07: per-loop timers and real stream cancellation.

The two limits to set before the next run: **10s connect / 30s receive on
Dio (with a 15s hydrate ceiling)** and **a 75s coach-stream inactivity
timeout**. Both are small diffs against well-tested seams.
