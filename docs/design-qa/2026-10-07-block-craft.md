# 2026-10-07 — Block craft: the frontend against the reference app

The user's request, as recorded in the
[plan](../plans/2026-10-07-frontend-block-craft/2026-10-07-frontend-block-craft.md)
it opened:

> optimize the frontend against the design language of the reference
> reimbursement app, keeping the cobalt blue hue family.

The reference (`~/Projects/Yalotein/reimbursement/app`) is gold-and-teal;
its hues were ignored entirely. What was ported is its **component
completeness**: measured fixed control heights, the three-state press
choreography, one shared modal chrome, table craft, and empty / loading /
error state discipline — rebuilt in the cobalt family on the frozen
Cobalt-Neo tokens.

## Decision

Run as the nine-wave block-craft plan. The primitive contract
(`client/lib/theme/block_*.dart`) shipped first and every surface was then
swept onto it; the token table and its test
(`client/test/cobalt_test.dart`) stayed frozen. Implementation record:
[changelog](../changelogs/2026-10-07-frontend-block-craft.md).

## What changed per surface

- **Shell + dock** — the top-right utility row became bordered
  `BlockIconButton`s on a cast block; dock items press via the shared
  grammar; the amber warn dot kept (its channel widened, below).
- **Auth door + landing** — forms moved to label-above `BlockField`s
  (44 fixed height, helper slots) and `BlockButton`s; brand discs,
  `InkReveal`/`InkStamp` motion, and the web-font gate preserved as
  identity exceptions.
- **Dashboard / workspace / ideas / settings** — every dialog onto the
  one 512px modal chrome (`showBlockModal` / `showBlockConfirm` /
  `showBlockForm`), raw error strings onto localized `BlockBanner` /
  `BlockError`, spinners onto the square `BlockLoader`, idea posters and
  board cards onto pressable blocks, read columns capped at 1120.
- **Project + roadmap** — node card dialogs, the view switcher and
  audience switchers unified on `BlockChoice`, minimap taps press the
  block, delete confirms on the shared chrome, status/kind enums
  localized.
- **PRD / coach / versions / revisions** — desk actions, commit dialog,
  revisions table, and export onto the primitives; audience selector
  collapsed onto the shared `BlockChoice`.
- **Warn channel** — amber `#FFD166` now covers warning-tone
  `BlockBanner`s and badges (dark theme; cobalt-700 on light grounds),
  not only the dock warn dot. Recorded as a
  [DESIGN.md](../../DESIGN.md) amendment.

## Trade-offs worth the user's eye

The Wave 7 fidelity review scored the result **7.5/10**; the honest gaps,
all filed as issues:

- **Keyboard users regressed** — the custom blocks have no focus ring and
  no Tab activation yet, where the Material buttons they replaced did
  ([issue](../issues/2026-10-08-block-focus-ring-keyboard.md)). This is
  the one high finding.
- Four overflow menus still open stock Material (rounded, blurred,
  ripple) popups; the workspace quick-capture FAB neither casts nor
  presses; coach API failures still collapse to the generic error
  string; dark-mode cobalt-950 bands are larger slabs than the decree
  intends — with six low findings beside them
  ([follow-ups](../issues/2026-10-08-block-craft-fidelity-followups.md)).
- The boot splash keeps its round spinner (plan-protected first-frame
  gate) and pull-to-refresh shows Material's round indicator — visible
  exceptions to "loaders are square blocks".
- The 390px web walk was not run; the roadmap utility cluster may crowd
  at that width.

Files touched: `client/lib/theme/block_*.dart` (new), `cobalt.dart`,
`cobalt_theme.dart`, `field_card.dart`, and ~40 files under
`client/lib/features/`, `client/lib/chrome/`, `client/lib/app.dart`,
`client/lib/l10n/`, plus `client/test/` — the full list is the
[changelog](../changelogs/2026-10-07-frontend-block-craft.md) and the
worktree diff.
