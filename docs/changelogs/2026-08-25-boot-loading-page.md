# 2026-08-25 — Boot loading page

The user asked for a loading page like `~/Projects/reimbursement`
(reference read from `reimbursement.tgz`): the Vue app renders
immediately, a full-screen centered spinner gates the router while
`checkSession()` runs, then the product appears. InsideOut's Flutter
`main()` instead **awaited** session + appearance + font hydration before
`runApp` — on a cold hosted load the user stared at a blank HTML page
for the whole session-restore round trip (network `/me` + refresh +
the web font gate).

## What changed

- `client/lib/session/boot.dart` (new): `BootController` — a
  ChangeNotifier gate that completes on success **or** error (a failed
  hydrate must lift the gate, not spin forever) — and `BootLoadingPage`:
  the 印 stamp over an ink ring spinner, deliberately text-free (locale
  may not be hydrated and the CJK fonts are not loaded yet).
- `main.dart`: `runApp` fires immediately; the four hydration tasks keep
  running in the background and `complete()` the gate on settle
  (`unawaited(Future.wait(...).then(complete, onError: complete))`).
- `app.dart`: while the gate is closed, a plain
  `MaterialApp(home: BootLoadingPage())` mounts — the router MaterialApp
  only mounts after boot settles, so **no product surface builds before
  the web fonts are ready** and the first-frame tofu fix
  ([2026-08-23](2026-08-23-web-font-flash.md)) is preserved.

## Verification

`flutter analyze` (no new issues; the 22 pre-existing lints unchanged)
and `flutter test` 50/50 — two new widget tests: the loading page shows
stamp + spinner and no text, and `InsideOutApp` gates on boot (loading
page and no product widgets until `complete()`, landing wordmark via
the router after).

Plan: [2026-08-25 boot loading page](../plans/2026-08-25-boot-loading-page.md).
