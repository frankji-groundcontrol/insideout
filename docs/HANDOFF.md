# HANDOFF

Living handoff for the next agent (Claude, Codex, human) taking over. Update
this on every checkpoint commit so a fresh agent can resume without re-reading
history. Keep it short and current.

## Current state

The InsideOut rewrite (Go backend + PostgreSQL/RLS + Nuxt 4 SSR frontend +
direct-Anthropic PRD Coach) is implemented and verified against the real
database and a real LLM endpoint. See
[docs/changelogs/2026-07-20-go-rewrite-and-rls-cutover/](changelogs/2026-07-20-go-rewrite-and-rls-cutover/index.md)
for what changed and how it was verified, and
[docs/architecture/](architecture/index.md) for the current system shape. The
old `juanleme` schema is gone (data migrated, then dropped).

On 2026-07-22 the product pivoted from "a PRD tool" to **idea → reality**:
branched-tree roadmaps on projects, GitHub progress sync, AI "build the MVP"
(PRD → generated roadmap + node expand). See
[docs/changelogs/2026-07-22-idea-to-reality/](changelogs/2026-07-22-idea-to-reality/index.md)
and the [plan](plans/2026-07-22-idea-to-reality.md).

On 2026-07-23 that day's Prisma cinematic theme detour was **reverted** to the
committed **国风留白 / Ink & Seal** world (celadon, sumi-ink, vermilion seal,
Noto Serif SC display; light + dark via `prefers-color-scheme`), and the public
landing was rebuilt as **「The Assembly」** — an ink build-instruction narrative.
Three a11y fixes landed from an adversarial critique (synchronous reduced-motion
gate, `BaseButton to` prop so CTAs aren't `<a><button>`, status-info chip
contrast). See
[docs/changelogs/2026-07-23-ink-seal-landing/](changelogs/2026-07-23-ink-seal-landing/index.md),
[PRODUCT.md](../PRODUCT.md) / [DESIGN.md](../DESIGN.md), and
[docs/design-system/CHANGELOG.md](design-system/CHANGELOG.md) `0.4.0`.

Also on 2026-07-23 the **whole authed frontend** was designed + implemented to
the Ink & Seal standard per the
[frontend-pages plan](plans/2026-07-23-frontend-pages.md): shared chrome
(`PageHeader` breadcrumbs, `BaseModal`, `BaseEmptyState`, restyled `NavBar` +
`error.vue`), workspace settings/members, PRD revisions reader, and the PRD
coach as a **~440px detachable right sidebar** with stage stepper, fact ledger,
and clickable suggested-prompt cards inline. Two follow-up directives landed the
same day: coach messages render real **markdown** (`utils/markdown.ts` +
`common/MarkdownBody.vue`, marked + dompurify, SSR-safe), and copy was reframed
from "build/code" to **idea-shaping + roadmap-definition** (the app writes no
code). See
[docs/changelogs/2026-07-23-coach-markdown-and-positioning.md](changelogs/2026-07-23-coach-markdown-and-positioning.md)
and [BUG-012](issues/2026-07-23-bug-012-project-list-null-latest-update-scan.md).

## In flight / next steps

- Nothing mid-flight. Open follow-ups live in [docs/TODO.md](TODO.md)
  ("Known Limitations": avatar upload placeholder, theme/locale in
  localStorage, plain-textarea PRD editors, GitHub sync is owner/admin +
  public-only, no roadmap drag-reorder UI) and [docs/issues/](issues/README.md).

## Key context / gotchas

- All work must be verified against a real database — see
  [docs/practices/](practices/README.md) and the RLS gotcha list in
  [docs/architecture/database-and-rls.md](architecture/database-and-rls.md).
- `DATABASE_URL` may point at a shared multi-tenant instance; never write
  outside the `insideout` schema.
- Bug records live in [docs/issues/](issues/README.md) as dated English
  files keeping their `BUG-NNN` identity (the former bilingual
  `docs/en/BUGS/` + `docs/cn/BUGS/` pair was retired on 2026-07-21 per user
  direction — docs are English-only now).

## How to verify

```
cd server && go build ./... && go vet ./... && go test ./...
# with a real DATABASE_URL in ../.env:
set -a && source ../.env && set +a && go test ./internal/store/... -run TestAuthz -v
cd ../app && pnpm test && npx nuxi typecheck && pnpm build
docker compose build
```
