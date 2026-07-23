# 2026-07-23 — Coach markdown rendering + idea-shaping positioning

Two corrections from a single directive: (1) coach conversation messages must
render as real markdown instead of raw `**`/`*` text, and (2) the app's copy
must stop implying it writes code — InsideOut does **idea-shaping and
milestone-definition**, not coding.

## What changed

- **Markdown rendering for coach messages.** New
  [`renderMarkdown`](../../app/src/utils/markdown.ts) util — `marked` (GFM,
  `breaks: true` for chat-style newlines) parses to HTML, then `dompurify`
  sanitizes it. New [`MarkdownBody.vue`](../../app/src/components/common/MarkdownBody.vue)
  renders the sanitized HTML via `v-html` (with an eslint justification) and
  styles the generated tags through `:deep()` against the existing
  `--color-*` design tokens: `color: inherit` so emphasis stays legible on
  both the light coach bubble and the dark user bubble, seal-underlined
  links, token-background inline-code chips (no monospace costume), neutral
  blockquote/table rules. SSR-safe: `DOMPurify.isSupported` is false on the
  server, so it degrades to escaped plain text; coach/user messages only
  render client-side (history loads in `onMounted`, streaming is
  interactive), so the sanitized path is the one that runs in practice.
  [`CoachPanel.vue`](../../app/src/components/prd/CoachPanel.vue) now renders
  both the message history and the live streaming bubble through
  `MarkdownBody` (caret kept alongside the stream).
- **Copy reframe: the app shapes ideas, it does not code.** Grounded in the
  backend truth — [`roadmap_ai.go`](../../server/internal/api/roadmap_ai.go)'s
  `handleBuildFromPrd` runs `planner.PlanMVP`, which generates a milestone
  roadmap tree and writes no code. An adversarial copy audit (workflow)
  separated genuinely-misleading claims from honest "your team builds / the
  app tracks" attribution. Five strings reframed per locale
  ([`en-US.ts`](../../app/src/i18n/locales/en-US.ts),
  [`zh-CN.ts`](../../app/src/i18n/locales/zh-CN.ts), parity preserved):
  - `heroTitle`: "Build your idea…" → "Shape your idea, piece by piece."
  - `ctaPrimary` / `ctaCloseButton`: "Start building" → "Start shaping"
  - `buildMVP`: "Build the MVP" → "Draft the roadmap" (zh: 规划路线图)
  - coach `finalize` suggestion: "Is this ready to build?" → "Is this ready
    to hand off to the team?"
  - Kept as honest: `buildingMVP` ("Drafting your roadmap…"), `step3Body`
    ("your team builds"), and all GitHub/commit tracking copy.
- **Backend fix recorded**: the workspace-board 500 from scanning a NULL
  `latest_update` into value types — see
  [BUG-012](../issues/2026-07-23-bug-012-project-list-null-latest-update-scan.md).
- **Craft floor**: the impeccable detector flagged the `6px` fallback inside
  `var(--radius-control, 6px)` as an off-scale radius; the token is always
  defined, so the dead fallback was dropped.

## Verification

- `npx nuxi typecheck` clean; `pnpm test` 20/20 (locale parity holds);
  `pnpm build` clean.
- Impeccable `detect.mjs` on the new/changed components: no findings.
- Browser, light + dark: landing hero reads "Shape your idea, piece by
  piece." with "Start shaping" CTAs; the PRD editor button reads "Draft the
  roadmap"; a live coach conversation renders `<strong>`/`<em>`/paragraphs
  with zero raw asterisks and correct contrast in both themes.

## Operator notes

No migration or config change. `marked` and `dompurify` are new runtime
dependencies (`@types/dompurify` intentionally not added — dompurify 3.x
bundles its own types).
