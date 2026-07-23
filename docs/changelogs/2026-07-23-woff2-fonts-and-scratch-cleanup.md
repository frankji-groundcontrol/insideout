# 2026-07-23 — WOFF2 fonts + scratch cleanup

Two small housekeeping changes that shrink the repo and the page payload, plus
removal of scratch left over from the Prisma detour.

## What changed

- **Converted the self-hosted PuHuiTi fonts to WOFF2.** The 4 weights
  (Regular 400, Medium 500, Bold 700, ExtraBold 800) were committed as full
  TTFs (~32 MB total) in the
  [frontend VC commit](2026-07-23-frontend-version-control.md). They are now
  WOFF2 (~16.4 MB total, ≈49–51% smaller per weight). `@font-face` in
  [`app/src/style.css`](../../app/src/style.css) now points at the `.woff2`
  files and the TTFs are deleted.
  - **Full glyph coverage kept — no CJK subsetting.** Users type arbitrary
    Chinese into ideas/PRDs, so dropping glyphs would risk tofu on
    user-generated content. The win comes purely from WOFF2's better
    compression (brotli), not from cutting the charset. Each font still covers
    Latin + full CJK (29,030 glyphs).
- **Deleted obsolete scratch.** Removed the `prisma-*.png` reference
  screenshots and the `.landing-mock/` mock left over from the reverted Prisma
  cinematic detour (see
  [the landing changelog](2026-07-23-ink-seal-landing/index.md)). None were
  referenced by any source. `.claude/` stays untracked (active local tool
  config).

## Verification

- WOFF2 files validated with fontTools: correct `woff2` flavor, 29,030 glyphs,
  full table set (glyf/cmap/GPOS/GSUB/vhea/vmtx…).
- Browser-verified against the running app: all four weights load and report
  `document.fonts.check(...) === true`; a CJK + Latin sample measures a width
  distinct from the `serif` fallback (proving the webfont, not a fallback,
  draws the glyphs). Light + dark and the EN + CN locales all render cleanly
  (screenshots taken).

## Operator note

The TTFs remain in git **history** (commit `2aea58e`), so full clones still
download the ~32 MB. Purging that requires a history rewrite
(`git filter-repo`) plus a **force-push** — destructive and outward-facing, so
it is **not** done here. Offered separately as an opt-in.
