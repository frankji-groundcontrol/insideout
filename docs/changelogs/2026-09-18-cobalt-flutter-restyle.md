# 2026-09-18 — Cobalt Flutter restyle

The live Flutter client wears the accepted Cobalt card lock. Ink & Seal
tokens, chops, and wax stamps are off signed-in and public surfaces.

## What changed

- Token type `Cobalt` replaces `InkSeal`
  ([`client/lib/theme/cobalt.dart`](../../client/lib/theme/cobalt.dart)).
- Signed-in chrome is a bottom dock
  ([`client/lib/chrome/product_dock.dart`](../../client/lib/chrome/product_dock.dart)).
- Lock surfaces: Home index + giant numeral, Ideas type wall + posters,
  PRD/Coach split desk, Commit fanned stack, Roadmap Now/Next/Later bands,
  Evidence cards on the project page.
- Landing, AuthDoor, and boot mark retokened; seal assets unused.
- [`DESIGN.md`](../../DESIGN.md),
  [`docs/architecture/frontend.md`](../architecture/frontend.md),
  design-system changelog **1.0.0**.

Plan: [`docs/plans/2026-09-18-cobalt-flutter.md`](../plans/2026-09-18-cobalt-flutter.md).

## Verification

```bash
(cd client && flutter analyze --no-fatal-infos && flutter test)
```

Browser walk: light/dark × zh-CN/en-US × Home, Ideas, PRD+Coach, Commit,
Roadmap, Evidence, theme toggle, dock.

## Operator

Ship the Flutter host after merge: `railway up --service app`. No server
or schema change.
