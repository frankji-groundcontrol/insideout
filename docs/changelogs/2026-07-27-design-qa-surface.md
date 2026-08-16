# 2026-07-27 — New docs/design-qa/ record surface

Adds a records surface for verbatim user design-QA feedback about how pages
look and about the frontend — each comment quoted exactly, with the resolution
it led to and the files touched — and a standing rule in both agent routers
that such comments get recorded there.

## Why

The user's standing instruction: comments about page appearance and the
frontend should be kept in a durable record, not lost to conversation. The
surface mirrors the existing changelogs/issues README structure (purpose,
convention, a dated-records index) — one dated file per QA thread.

## What changed

- **`docs/design-qa/README.md` (new).** Defines the surface and its convention
  (one dated file per thread; each entry quotes the user verbatim, records the
  decision/resolution, lists affected files) with an empty records index.
- **`docs/index.md`.** Lists `design-qa/` under Records, in the existing entry
  style.
- **`CLAUDE.md` / `AGENTS.md`.** Identical standing rule appended to their
  respective "Recording changes" / "Docs and records" sections: when the user
  posts comments about how a page looks or the frontend, append them to the
  relevant dated file under `docs/design-qa/` (creating one if needed) and keep
  its README index current.
- **Seed record:** [design-qa/2026-07-27-auth-door.md](../design-qa/2026-07-27-auth-door.md)
  — the auth-door QA thread that this change shipped alongside; implementation
  detail in [the auth-door changelog](2026-07-27-auth-door-modal.md).

## Verification

All relative links resolve (`test -e` from repo root): the new README, its
changelog cross-link, and the references from `docs/index.md`, `CLAUDE.md`, and
`AGENTS.md`. Both routers state the identical rule sentence. Docs-only change;
no code or build impact.

## Operator notes

None — documentation structure only.
