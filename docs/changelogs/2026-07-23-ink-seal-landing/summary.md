# Summary — what changed, phase by phase

The work ran in three phases from [the plan](../../plans/2026-07-23-ink-seal-landing.md):
**A** reconcile the theme, **B** record the world, **C** rethink the landing.
This was a design/frontend change only — no backend, schema, or API changes.

## Phase A — theme reconciliation (revert Prisma → Ink & Seal)

The 2026-07-22 Prisma re-theme (pure black + warm cream, dark default) was an
uncommitted detour. Reverted to the committed Ink & Seal world while keeping
yesterday's real product work (roadmap tree, GitHub sync, AI MVP builder,
semantic-token component migration).

- [`app/src/assets/tokens.css`](../../../app/src/assets/tokens.css) — restored the
  HEAD Ink & Seal values (celadon/ink ground, vermilion `seal`, `carve`,
  ink `btn`/`btn-fg` that inverts to paper on dark, the five `status-*` ramps).
  Token _keys_ unchanged — one revert re-themes every semantic consumer.
- [`app/tailwind.config.js`](../../../app/tailwind.config.js) — `fontFamily.serif`
  leads with **Noto Serif SC** (the Song/Mincho display face); sans stays on
  self-hosted **Alibaba PuHuiTi**.
- [`app/src/style.css`](../../../app/src/style.css) — kept the PuHuiTi `@font-face`;
  dropped the Prisma noise utilities + stale comments.
- [`app/nuxt.config.ts`](../../../app/nuxt.config.ts) — the FOUC default reverted from
  forced-dark back to `prefers-color-scheme`; kept the Go-proxy, tokens.css-load
  and InsideOut-title work; added the Noto Serif SC display link.

## Phase B — record the world

- [`PRODUCT.md`](../../../PRODUCT.md) — Brand Commitments now name Ink & Seal as the
  binding whole-product identity (replacing the stale "open decision" line).
- [`DESIGN.md`](../../../DESIGN.md) — the Ink & Seal world documented whole-product
  (tokens, typography, the seal accent discipline, light + dark).

## Phase C — landing rethink (Persuade mode)

Direction confirmed with the user as **「The Assembly」/「一步步」**. New components
under `app/src/components/landing/`:

- `AssemblyDiagram.vue` — the signature device: an inline-SVG build path of four
  ink icons (spark → doc → tree → seal) joined by dashed guide arrows. `hero`
  mode plays the click-in sequence via CSS keyframes (the seal stamps down
  last); `progress` mode is reused as each step's "you are here" mini-map
  (dim past / solid present / ghosted future).
- `AssemblyHero.vue` — the first viewport: celadon field, the Song-serif claim,
  the two CTAs, and the diagram assembling itself below.
- `AssemblyStep.vue` — one numbered build-instruction per step, with the zh step
  chop (落墨 / 成文 / 分枝 / 盖印) and the recurring mini-map.
- `StepPeek.vue` — the 1:1 call-out box: an honest skeleton peek at the real
  artifact each step produces (idea inbox card, coach interview, branched
  roadmap with live statuses, GitHub-commit seal timeline).
- [`app/src/pages/index.vue`](../../../app/src/pages/index.vue) — assembles hero + four
  steps + a "press your first seal" close; `definePageMeta({ public: true })`.
- i18n — all copy in `app/src/i18n/locales/{en-US,zh-CN}.ts` under `landing.*`.

### Accessibility fixes (Ultracode adversarial critique)

A multi-agent critique (4 critics → per-finding adversarial verify) confirmed
three majors; all fixed:

1. **Reduced-motion gate** — `app/src/composables/useReducedMotion.ts` initialized
   `ref(false)` and only read `matchMedia` in `onMounted`, but motion-v resolves
   each component's `initial` once at construction, so the entrance animation
   ran anyway for reduced-motion users. Now initialized **synchronously** on the
   client (`import.meta.client ? matchMedia(...).matches : false`), keeping the
   live `change` listener.
2. **Nested interactive** — the CTAs were `<NuxtLink><BaseButton></NuxtLink>` →
   `<a><button>`. `BaseButton` gained a **`to` prop** that renders a single
   styled `NuxtLink` when present; applied at the hero and close CTAs.
3. **Status-chip contrast** — the roadmap `in_progress` chip was `text-seal` on
   `bg-seal/15` (3.53:1 light / 3.22:1 dark, below 4.5:1). Switched to the design
   system's purpose-built pair `bg-status-info-bg text-status-info-fg` (~5.0:1
   light / ~6.2:1 dark), matching the sibling done/locked chips and `BaseBadge`.

The step chop (`text-carve` on `bg-seal`, 4.48:1) was adjudicated **intentional**:
it is `aria-hidden="true"` decorative and its meaning is redundant with the
adjacent step number, so it stays as the brand signature.

### Out of scope (recorded, not fixed here)

- **BUG-011** — locale SSR hydration mismatch (server renders `zh-CN`, client
  hydrates to saved `en-US`). Pre-existing and app-wide; recorded in
  [`docs/issues/`](../../issues/README.md) for a dedicated i18n-SSR pass.
