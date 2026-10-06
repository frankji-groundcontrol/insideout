# 23 · Reliability Ops

**Lens:** Timeouts, retries, offline behavior, failure surfaces, SSE robustness
**Score:** **5.5 / 10**
**Bottom line:** The api/session core is better than average — the single-flight refresh is correct under concurrency and loopback-tested, SSE parsing is unit-tested, the boot gate completes on error, and live-stream retries are bounded. But the client has NO timeout on any network call anywhere, which means the boot gate, the coach composer, and every data page can hang silently forever; an offline app start deletes the user's tokens; and three error surfaces either render raw English or structurally always fall through to the generic apology. All fixable with small, targeted diffs.

Reviewed branch `main` at `f185e94`; a skeptic pass re-checked every Critical/High finding — corrections marked [Correction]. Screenshot evidence lives in `./shots/`.

---

## Scope

`client/lib/api` (client.dart, sse.dart, sse_transport_web.dart, errors.dart, errors_l10n.dart), `client/lib/session` (token_store.dart, session.dart, boot.dart), `features/prd/coach_panel.dart`, `features/project/roadmap_page.dart`, and the error surfaces on dashboard_page.dart and versions_page.dart. The question this pass asks: if the network misbehaves, does the client degrade predictably — and the honest answer is "sometimes, and the worst cases are the silent ones."

## What is good

- `client/lib/api/client.dart:77-83` — single-flight refresh via `_refreshing ??= refresh().whenComplete(...)` is exactly right, and client/test/api_token_refresh_test.dart proves it against a real loopback HTTP server, including the concurrent-401 case where two simultaneous me() calls share one refresh.
- `client/lib/api/sse.dart:92-94` — decodeSseBytes copies chunks before running the utf8 transformer, the correct fix for multibyte characters split across web ReadableStream chunks; client/test/sse_test.dart covers the parse boundary.
- `client/lib/main.dart:27-34` — the boot gate lifts on error as well as success (`.then((_) => boot.complete(), onError: (_) => boot.complete())`), so a failed hydrate cannot wedge the app behind BootLoadingPage — at least when the failure is an error rather than a hang.
- `client/lib/features/project/roadmap_page.dart:56-95` — both live-stream loops cap backoff at 5s and treat 401/403/404 as fatal, so a deleted project or dead session does not spin the retry wheel forever.
- `client/lib/api/errors_l10n.dart` — one mapper owns the status→localized-copy contract and is directly unit-tested; most pages route through it, which is why the two violations stand out so visibly.

## Findings, worst first

### rel-no-network-timeouts — High

No network call anywhere has a timeout, so a black-holed connection hangs the boot gate forever and the onError path never fires. `Session.hydrate()` calls `_me()` and, on failure, `_refresh()` — both unbounded; `main.dart`'s `Future.wait(...).then(..., onError: boot.complete)` handles *errors*, but a black-holed connection (captive portal, firewall DROP rule) is neither resolution nor rejection. The app sits on `BootLoadingPage` indefinitely. The same absence means the coach composer can lock forever (`sending` stays true, text field disabled at `coach_panel.dart:212`) and every FutureBuilder data page can spin forever. Tellingly, `errors_l10n.dart:38` has a `TimeoutException` branch and `errors_l10n_test.dart:33` asserts it — a tested contract the runtime cannot currently produce.

Evidence: client/lib/api/client.dart:30

```
_dio = Dio(BaseOptions(baseUrl: baseUrl, headers: {'Content-Type': 'application/json'}, validateStatus: (s) => s != null && s < 500)); — a grep for connectTimeout / receiveTimeout / sendTimeout across client/lib returns only web-font loading timeouts.
```

**Fix:** Set connectTimeout (10s) and receiveTimeout (30s) on Dio BaseOptions, and wrap hydrate's network steps in an explicit timeout so the gate lifts into the signed-out state instead of spinning.

**Verdict:** CONFIRMED — client.dart:30-34 constructs Dio() with no connect/receive/send timeout; the only .timeout( in lib is web_fonts_web.dart:46; main.dart:33 boot gate awaits a future that never completes when the API hangs.

### rel-hydrate-wipes-tokens — High

`hydrate()` deletes stored tokens on ANY failure, so an offline or flaky app start silently logs the user out. `session.dart:52-60` nests `_me()` → `_refresh()` → re-`_me()`, and the outer `catch (_) { user = null; await clear(); }` deletes both tokens for *any* exception — a 401, but also a `SocketException` on a commuter train. The result: an offline app start silently wipes the session, and the user re-authenticates (magic-link email and all) for what was a dead tunnel, not a dead session.

Evidence: client/lib/session/session.dart:58

```
try { user = await _me(); } catch (_) { await _refresh(); user = await _me(); } } catch (_) { user = null; await clear(); } — where clear() calls _store.delete().
```

**Fix:** Only call clear() when the failure is an ApiException with status 401/403; on transport errors keep the token pair and leave the user signed out for this launch without deleting storage.

**Verdict:** CONFIRMED — session.dart:58-60 catch (_) { user = null; await clear(); } is untyped; clear() (:123-127) deletes stored tokens, so an offline app start silently logs the user out.

### rel-coach-stream-no-timeout — High

A coach SSE stream that stalls mid-response locks the composer forever — no inactivity timeout, no cancel control. If the server accepts the coach POST and the LLM upstream hangs (or a proxy holds the connection open), no frames arrive, the stream never closes, `sending` stays true, and the text field plus send button are disabled with only the "thinking" hint — permanently for that panel. The user must leave the page and come back.

Evidence: client/lib/features/prd/coach_panel.dart:212

```
enabled: !sending && retry == 0; — client/lib/api/sse_transport_web.dart:27 `final res = await web.window.fetch(url.toJS, init).toDart;` with no AbortController or deadline; client/lib/api/client.dart:502 `await readCoachSse(chunks, handlers);` waits until the connection closes.
```

**Fix:** Add an inactivity timeout to sendCoach (e.g., abort if no frame arrives for 75s) using a Timer that completes/cancels the stream, and surface a localized error that unlocks the composer.

**Verdict:** CONFIRMED — client.dart:502 readCoachSse has no deadline; sse_transport_web.dart:27 has no AbortController; coach_panel.dart:212 enabled: !sending && retry == 0 leaves the composer permanently disabled while a hung stream holds sending=true.

### rel-coach-error-string-sniffing — Medium

The coach panel stores the exception's raw English message, so its error line always renders the generic apology and its retry label is chosen by sniffing the English substring 'unavailable'. The panel stores `e.message` (a String), then hands that string to `localizedError` (`coach_panel.dart:194`), which matches no typed branch and *always* returns `error.generic` — the real failure classification is discarded before it reaches the UI. Meanwhile the countdown label is picked by message-text sniffing, and the server's permit-queue 503 says "AI service is busy, try again shortly" (`server/internal/agent/coach.go:150`) — no 'unavailable' substring — so a circuit event is labeled "rate limited", and any server that changes or localizes its message text silently flips the label.

Evidence: client/lib/features/prd/coach_panel.dart:96

```
setState(() => error = e.message); — line 194 Text(localizedError(error!, l10n.t), ...) is handed a String, which matches no ApiException/DioException branch in errors_l10n.dart:16-41 and always returns `error.generic`; line 199 l10n.t(error != null && error!.contains('unavailable') ? 'coach.serviceUnavailable' : 'coach.rateLimited', ...). Server side the permit-queue 503 is "AI service is busy, try again shortly" (server/internal/agent/coach.go:150) — no 'unavailable' substring.
```

**Fix:** Keep the ApiException object in state and pass it to localizedError; pick the retry label from `e.code == 'CIRCUIT_OPEN'` (or status 503), never from message text.

### rel-shared-retry-timer — Medium

One shared `_retryTimer` field backs both the progress and presence backoff loops, so the second `_backoff` clobbers the first loop's timer handle and dispose can cancel at most one of two pending timers. Both loops await `_backoff` (`roadmap_page.dart:69` and `:92`); the second assignment discards the first timer's reference. The mounted checks make this mostly harmless today, but it is exactly the kind of shared-mutable-handle bug that bites the next edit, and the orphaned completer garbage lingers.

Evidence: client/lib/features/project/roadmap_page.dart:40

```
_retryTimer = Timer(d, done.complete); (plain assignment, no per-loop handle); line 99 _retryTimer?.cancel(); in dispose; both loops await _backoff at lines 69 and 92.
```

**Fix:** Give each loop its own Timer local, or track timers in a set and cancel them all in dispose; keep the completer completing via `Timer(d, done.complete)` per call.

### rel-sse-streams-never-cancelled — Medium

Live SSE streams are never cancelled when the roadmap page disposes — the web reader is never cancelled and dio gets no CancelToken — so each page visit leaks a suspended loop and an open connection until the server drops it. The progress/presence loops only re-check `mounted` after the current stream returns, which, for a healthy connection, is never. Every roadmap visit parks suspended loops and open connections until the server or the process dies.

Evidence: client/lib/api/sse_transport_web.dart:35

```
_readChunks loops `await reader.read()` with no `reader.cancel()` path; client/lib/api/client.dart:505-508 comment: "dio has no per-request cancel here, so the page closes by letting it die with dispose"; roadmap_page.dart:110-111 fire-and-forget loops that only exit `while (mounted)` after the current stream returns.
```

**Fix:** Thread a cancellation signal (CancelToken for dio, a cancel callback that calls reader.cancel() for web fetch) from the page's dispose into both stream loops so the awaits unwind promptly.

### rel-coach-setstate-after-dispose — Medium

Coach stream callbacks and catch blocks call setState with no mounted guard, so navigating away mid-stream throws setState-after-dispose. `onDelta` and both catch blocks call `setState` unguarded; only the `finally` and the fact/snack callbacks check mounted. The sibling callbacks got the guard; these three did not.

Evidence: client/lib/features/prd/coach_panel.dart:80

```
onDelta: (chunk) => setState(() => stream += chunk); — lines 96 and 99 setState(() => error = e.message) / setState(() => error = e.toString()) in catch blocks. Only the `finally` (line 101) and the fact/snack callbacks (lines 84, 90) check mounted.
```

**Fix:** Guard every stream callback and catch-block setState with `if (!mounted) return;` (or the setState(() {}) pattern already used in the finally).

### rel-raw-error-pages — Medium

Dashboard and versions pages render the raw exception text as page content, bypassing the localizedError standard every other page follows. `'${snap.error}'` becomes the entire page body in a zh-CN-first product, untranslated, with no retry affordance — a broken window next to roadmap_page, which routes through `localizedError` correctly (`roadmap_page.dart:211/220`).

Evidence: client/lib/features/dashboard/dashboard_page.dart:98

```
if (snap.hasError) return Center(child: Text('${snap.error}')); — client/lib/features/prd/versions_page.dart:73 identical line. Compare roadmap_page.dart:211/220 which use `localizedError(snap.error!, l10n.t)`.
```

**Fix:** Route both branches through localizedError(snap.error!, l10n.t) and add a localized retry button that re-creates the FutureBuilder's future.

### rel-5xx-misclassified-network — Low

HTTP 5xx responses are classified as "network" errors because dio's validateStatus turns them into DioException before the mapper can read the status, making the error.server branch unreachable for normal calls. During a server deploy every page tells the user a network problem on their side ("check your connection") instead of a server outage — sending them to debug Wi-Fi that was never the issue and masking real incidents from support.

Evidence: client/lib/api/errors_l10n.dart:38

```
validateStatus: (s) => s != null && s < 500; — if (error is DioException || error is TimeoutException) { return t('error.network'); } — the error.server branch at lines 33-35 only triggers for ApiException, which plain HTTP 5xx never becomes.
```

**Fix:** In localizedError, inspect `DioException.response?.statusCode` before the generic DioException branch and map >=500 to error.server.

### rel-web-token-storage — Low

Web builds store both auth tokens through flutter_secure_storage, which on the web has no secure enclave and persists values in localStorage — readable by any same-origin script. Refresh-token rotation bounds the exposure window but does not remove it; this is a documented-the-web-trade-off or move-to-HttpOnly-cookie decision, not a code bug, but it should be written down in the architecture docs rather than left implicit.

Evidence: client/lib/session/session.dart:18

```
_store = store ?? SecureTokenStore() with no kIsWeb branch; client/pubspec.yaml:40 `flutter_secure_storage: ^9.2.4` (web implementation degrades to localStorage; encryption is not supported there).
```

**Direction:** Accept and document the web trade-off with refresh-token rotation as the mitigation and short access-token TTL, or move web sessions to an HttpOnly cookie + SameSite=Lax model.

**First check:** Confirm the deployed threat model treats XSS as out of scope or not (docs/architecture/), then record the decision where token storage is documented.

### rel-timeout-branch-untestable — Low

The TimeoutException branch of localizedError is untestable in production because nothing in the client can throw one — there are no timeouts anywhere. The test suite green-lights a timeout contract the runtime cannot deliver; a future reader assumes hangs are already handled and skips adding real timeouts (the rel-no-network-timeouts gap).

Evidence: client/lib/api/errors_l10n.dart:38

```
if (error is DioException || error is TimeoutException); client/test/errors_l10n_test.dart:33 `expect(key(TimeoutException('t')), 'error.network');` — no timeout configuration exists in client/lib.
```

**Fix:** Land the Dio/hydrate timeouts (which make Dio throw TimeoutException), keeping this branch and its test honest; until then note the branch is aspirational.

### rel-brightness-by-color-identity — Low

Brightness is detected by comparing the resolved surface color to a constant instead of asking the theme, an identity check that breaks the moment any theme variant reuses the same ground color. This also intersects the neo-brutalist pivot: a four-step near-black ladder with more than one dark ground hex will silently flip coach-panel polarity.

Evidence: client/lib/features/prd/coach_panel.dart:231

```
extension on Cobalt { bool get brightnessIsDark => surfaceBase == Cobalt.darkGround; }
```

**Fix:** Use `Theme.of(context).brightness` (or pass Brightness down from the build method) instead of comparing colors.

## Recommendations

- Set explicit Dio timeouts — connectTimeout 10s, receiveTimeout 30s — in ApiClient's BaseOptions, and wrap Session.hydrate's network steps in a 15s timeout so the boot gate always lifts.
- In Session.hydrate, only clear stored tokens on ApiException 401/403; on transport failures keep the token pair and present a signed-out state for this launch without deleting storage.
- Add a frame-inactivity timeout (about 75s) and a cancel path to sendCoach so a stalled coach stream releases the composer instead of locking it.
- Store the ApiException object in CoachPanel state, pass it to localizedError, and derive the retry label from e.code/status rather than English message substrings.
- Give each roadmap retry loop its own Timer handle and thread cancellation (dio CancelToken, web reader.cancel()) into progressStream and presenceStream so dispose actually tears the streams down.
- Route dashboard_page.dart:98 and versions_page.dart:73 through localizedError and give both pages a localized retry action.
- Map DioException carrying a >=500 response status to error.server in localizedError so server outages stop masquerading as user network failures.
- Add mounted guards to CoachPanel's onDelta and catch-block setState calls, and record the web localStorage token-storage trade-off in the architecture docs with rotation as the stated mitigation.

## What this role would do first

Set the two limits before the next run, both small diffs against well-tested seams: add `connectTimeout: 10s` / `receiveTimeout: 30s` to the `BaseOptions` in `client/lib/api/client.dart:30` plus a ~15s timeout around the network steps of `Session.hydrate()` in `client/lib/session/session.dart:52` so nothing can hang the boot gate, then add a 75-second frame-inactivity timeout and cancel path to `sendCoach` in `client/lib/features/prd/coach_panel.dart` so a stalled SSE response cannot lock the composer.
