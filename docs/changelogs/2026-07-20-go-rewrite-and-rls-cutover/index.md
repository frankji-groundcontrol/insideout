# 2026-07-20 — Go Rewrite and RLS Cutover

Large change record: the backend was rewritten from Supabase edge functions
(TypeScript/Deno) to a Go service, the frontend was switched to talk only to
that Go API, JWT+RLS defense-in-depth was added at the database level, and the
old `juanleme` schema's real data was migrated into `insideout` and the old
schema dropped. The product was rebranded from JuanLeMe (卷了么) to InsideOut.

## Contents

- [summary.md](summary.md) — what changed, subsystem by subsystem.
- [verification.md](verification.md) — the verification actually performed,
  and what remains open.
- [migration-notes.md](migration-notes.md) — what operators of an old
  JuanLeMe deployment need to know.

## Primary sources

- [Rewrite plan](../../plans/2026-07-20-go-rewrite/README.md) (decisions D1–D10)
- [Progress checklist](../../TODO.md) (phases P1–P7)
- Bug book: [docs/issues/](../../issues/2026-07-20-bug-001-happydom-localstorage.md)
  entries BUG-001 through BUG-010
