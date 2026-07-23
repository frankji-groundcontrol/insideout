# Changelogs

Dated records of what changed in this repository, written for maintainers.

## Convention

- **Ordinary change**: one dated file, `YYYY-MM-DD-short-slug.md`.
- **Large change** (multi-phase, multiple subsystems): a dated folder,
  `YYYY-MM-DD-short-slug/`, with an `index.md` linking focused child files
  (e.g. `summary.md`, `verification.md`, `migration-notes.md`).

Each record states what changed, how it was verified, and what an operator of
an existing deployment needs to do. Cite sources (plans, bug book entries,
code) with relative links.

## Records

- [2026-07-23 — Bring the frontend rewrite + infra under version control](2026-07-23-frontend-version-control.md)
  — tracked the Nuxt 4 SSR frontend rewrite and deployment infra
  (docker-compose + Dockerfiles), removed the superseded `supabase/` backend,
  and self-hosted the PuHuiTi fonts. Completes the cutover: the whole new
  stack is now in git.
- [2026-07-23 — Coach markdown rendering + idea-shaping positioning](2026-07-23-coach-markdown-and-positioning.md)
  — coach messages render real markdown (marked + dompurify, SSR-safe) via a
  token-styled `MarkdownBody`, and copy was reframed from "build/code" to
  idea-shaping + roadmap-definition. Also records BUG-012 (workspace-board
  NULL-scan 500).
- [2026-07-23 — Ink & Seal reconciliation + landing rethink](2026-07-23-ink-seal-landing/index.md)
  — reverted the Prisma cinematic detour back to the committed Ink & Seal world,
  and rebuilt the public landing as 「The Assembly」 (ink build-instructions;
  three a11y fixes from an adversarial critique).
- [2026-07-22 — Idea → Reality](2026-07-22-idea-to-reality/index.md)
  — branched-tree roadmap on projects, GitHub progress sync, AI "build the
  MVP" (PRD → generated roadmap), and a full UI re-theme to the Prisma
  cinematic reference.
- [2026-07-21 — Documentation reorganization](2026-07-21-docs-reorganization.md)
  — clean-repo-org layout: routers, architecture/usage/learning/practices
  surfaces, bug book merged into `docs/issues/` (English-only), legacy
  retired to `docs/history/`, guardrail installed.
- [2026-07-20 — Go rewrite and RLS cutover](2026-07-20-go-rewrite-and-rls-cutover/index.md)
  — full backend rewrite to Go, frontend swap to the Go API, JWT+RLS
  defense-in-depth, and the juanleme → insideout data cutover.
