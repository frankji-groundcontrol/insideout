# Plans

Live and historical task plans for this repository. Each plan states scope,
decisions, and status; update the plan in place as work proceeds rather than
narrating progress only here.

## Active

- [2026-07-23 — Comprehensive frontend: page structure + design](2026-07-23-frontend-pages.md)
  — whole-app IA (global chrome, workspace settings/members, PRD revisions), a
  detachable CoachPanel with inline suggested-prompt cards, and every page
  brought to the Ink & Seal standard. Status: in progress.
- [2026-07-21 — PRD agent harness redesign](2026-07-21-prd-agent-harness/index.md)
  — research-backed design for the PRD coaching harness (fact ledger,
  mechanical stage gates, fresh-context critic, dispatch/queueing/failure
  hardening), reviewed via gstack plan-ceo-review + plan-eng-review.
  Status: reviewed, ready to implement (pending plan §9 user decisions).

## Completed

- [2026-07-23 — Ink & Seal reconciliation + landing rethink](2026-07-23-ink-seal-landing.md)
  — reverted the Prisma detour to the committed Ink & Seal world, recorded it in
  PRODUCT.md/DESIGN.md, and rebuilt the landing as 「The Assembly」 (plus three
  a11y fixes from an adversarial critique). Status: complete.
- [2026-07-22 — Idea → Reality](2026-07-22-idea-to-reality.md)
  — branched-tree roadmap, GitHub progress sync, AI "build the MVP", and the
  (since-reverted) Prisma UI re-theme. Status: implemented.
- [2026-07-21 — Reorganize repo docs per clean-repo-org](2026-07-21-clean-repo-org.md)
  — thin router files, modular `docs/` records, a 23-agent staleness audit
  of every doc against the code, bug book merged into `docs/issues/`
  (English-only), guardrail installed. Status: complete.

## Historical

- [2026-07-20 — Go rewrite](2026-07-20-go-rewrite/README.md) — the backend
  rewrite from Supabase/TypeScript to Go + direct PostgreSQL, JWT+RLS
  defense-in-depth, the `juanleme` real-data cutover, and the langchaingo
  removal. This is also this repo's primary current-state architecture
  source until [`docs/architecture/`](../architecture/index.md) fully absorbs
  it. Status: implemented; a few DB-gated items remain open (see the plan's
  own phase table and [`docs/TODO.md`](../TODO.md)).
