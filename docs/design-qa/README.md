# Design QA

Verbatim user design-QA feedback about how pages look and about the
frontend — what the user said, the decision or resolution each comment led
to, and the files it touched. Dated records, written for maintainers;
English only.

## Convention

- **One dated file per QA thread**, `YYYY-MM-DD-short-slug.md` — a thread is
  one sitting's worth of related comments (e.g. one page or one visual
  concern), not one comment.
- Each entry **quotes the user's comment verbatim**, records the
  decision/resolution it led to, and **lists the affected files** (with
  relative links where useful).
- Cite the resulting implementation with relative links into
  [changelogs](../changelogs/README.md) or the code; keep the standing
  record here, not the fix detail.

## Records

<!-- Entries are added here as QA threads accumulate; newest first. -->

- [2026-09-17 — Recap (3Q) design language](2026-09-17-3q-recap-language.md)
  — user voted 3Q; OpenDesign reference + tokens; images first.
- [2026-09-17 — Vibrant for younger audiences](2026-09-17-vibrant-younger-audiences.md)
  — vibrant / no Ink & Seal / vote 3 / 3Q lock; Flutter waits.
- [2026-08-24 — Login form action stack](2026-08-24-login-form-action-stack.md)
  — "so ugly"; four identical stacked red links under 登录 re-grouped into
  secondary sign-in buttons + a footer row.
- [2026-08-23 — Web first-refresh font flash](2026-08-23-web-font-flash.md)
  — "it shows garbage characters for a second before the Chinese finally
  appears"; CanvasKit first frame gated on CJK font readiness.
- [2026-08-19 — Flutter cutover dropped Ink & Seal](2026-08-19-restore-ink-seal.md)
  — "ditched our visual on nuxt" / "an infra migration does not mean visual
  change"; Material 3 lock reversed; tokens + AuthDoor chrome restoring on
  Flutter.
- [2026-07-27 — Auth door: login/register as a prompted floating modal](2026-07-27-auth-door.md)
  — two comments ("ugly… the seal logo is not applied" / "make it a prompted
  floating modal with motion?") resolved by one shared `AuthDoor` dialog shell;
  implementation in the
  [auth-door changelog](../changelogs/2026-07-27-auth-door-modal.md).
