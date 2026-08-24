# 2026-08-25 — Boot loading page (reimbursement pattern)

Status: **Finished** — implemented and verified; see the
[changelog](../changelogs/2026-08-25-boot-loading-page.md).

## Why

`main()` awaited session + appearance + fonts hydration **before**
`runApp`, so a cold hosted load showed a blank HTML page for the whole
session-restore round trip (plus the web font gate). The user asked for
a loading page like `~/Projects/reimbursement`: the app renders
immediately, a full-screen centered spinner gates the router while
session restore runs, then the product appears.

## Design

- `session/boot.dart`: `BootController` (ChangeNotifier, done/complete)
  + `BootLoadingPage` — 印 stamp + ink ring spinner, no text that needs
  fonts.
- `main.dart`: `runApp` immediately; the four hydration tasks keep
  running in the background and `complete()` the boot gate on settle
  (success or error — a hung boot must not spin forever silently).
- `app.dart`: while `!boot.done`, a plain `MaterialApp(home:
  BootLoadingPage())` — nothing product-ish mounts before fonts, so the
  web tofu-flash fix is preserved (the router MaterialApp only mounts
  after the gate lifts).
- Widget tests: loading page renders; the app flips to the router when
  the gate completes.

## Out of scope

- Route-level skeletons (per-page loading states) — this is the boot
  gate only.
- Native launch screens (OS-provided already).

## Checklist

- [x] Plan + board row
- [x] boot.dart + main/app rewiring
- [x] Widget tests; analyze + full suite green (50/50)
- [x] Changelog + indexes; commit, push, deploy app
