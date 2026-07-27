# 2026-07-27 — PuHuiTi fonts: reference, don't redistribute

The Alibaba PuHuiTi web-font binaries are removed from the repository and from
git history. The font is now **referenced by name only**, with proper
attribution, instead of being bundled. Supersedes the
[2026-07-23 self-hosting](2026-07-23-woff2-fonts-and-scratch-cleanup.md) of the
same font.

## Why

Alibaba PuHuiTi 2.0 is published by Alibaba for free personal and commercial
**use**, but it is **not** an open-source (OSI/OFL) license — the font file is
marked "All rights reserved" (© 2020-2021 Alibaba (China) Co., Ltd.), trademark
Alibaba Group Holding Limited, made by Alibaba Design + Hanyi Fonts, embedding
permission Preview & Print (OS/2 `fsType=4`). "Free to use" is **not** "free to
redistribute the font file," so committing ~16.8 MB of the font binary to a
public repository (HEAD or history) is redistribution that exceeds the grant.
The correct posture is to *refer to* the font, not host it. (An earlier Git-LFS
plan was abandoned for this reason: LFS still hosts the file publicly.)

## What changed

- **Deleted the 4 WOFF2 binaries** (`app/public/fonts/AlibabaPuHuiTi-*.woff2`,
  ~4 MB each) from the working tree, and **purged them from git history** with
  `git filter-repo --invert-paths --path app/public/fonts` so no commit ships
  the file. `main` was force-pushed (lease-pinned).
- **`app/src/style.css`** — removed the four PuHuiTi `@font-face` blocks (there
  is no longer a file to serve); the header comment now states the
  reference-only policy and points at the attribution.
- **`app/tailwind.config.js` / `app/nuxt.config.ts`** — kept `"Alibaba PuHuiTi"`
  first in the `sans` stack (the reference: a visitor who has it installed
  renders it) and updated the comments that claimed the font was self-hosted.
  Rendering now falls through to `Noto Sans SC` (already loaded from Google
  Fonts) and the platform CJK face (PingFang SC / Microsoft YaHei) — the stack
  was already designed for this. The display serif (Noto Serif SC) was already a
  Google-Fonts reference, so this makes the body face consistent with it.
- **Attribution** — new bilingual "Third-party assets / Fonts" section in
  [`README.md`](../../README.md): copyright, trademark, manufacturer, version,
  the official source (<https://alibabafont.taobao.com/>), and an honest
  license note ("free to use, not an open-source license; we redistribute no
  font file").
- **Docs** — corrected the "self-hosted PuHuiTi" claims in
  [`DESIGN.md`](../../DESIGN.md) and [`PRODUCT.md`](../../PRODUCT.md). Dated
  records that described the 2026-07-23 self-hosting were left untouched — they
  are accurate for their date; this entry is the superseding record.

## Verification

- `git ls-files` shows no font binaries; `git log --all` no longer contains
  `app/public/fonts` after the rewrite.
- `cd server && go build ./... && go vet ./...` green (unchanged backend).
- `cd app && npx nuxi typecheck && pnpm build` green — removing the `@font-face`
  `url()` references and the `public/fonts/` files breaks nothing, because those
  URLs were absolute `/public` paths (never bundled by Vite) and the sans stack
  always had a CJK fallback.

## Operator notes

- **Anyone with a clone must re-clone or hard-reset** — the history rewrite
  changed commit SHAs: `git fetch origin && git reset --hard origin/main`, or a
  fresh clone.
- **The app no longer ships the font.** Deployed instances render body text in
  Noto Sans SC / the platform CJK face unless the visitor has PuHuiTi installed.
  To reproduce the exact Ink & Seal body look in a given environment, obtain and
  install PuHuiTi from the official source yourself (that is your own permitted
  "use"); the app intentionally does not fetch or bundle it.
