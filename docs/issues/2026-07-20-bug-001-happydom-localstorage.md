# BUG-001: happy-dom 20's `localStorage` stub is missing `.clear()`, breaking the null-check fallback

**Found**: 2026-07-20, during the InsideOut rewrite (P5), running the pre-existing frontend test suite as a baseline before making any changes.

**Symptom**: `TypeError: localStorage.clear is not a function` in every test file whose `beforeEach` called `localStorage.clear()` (12 files, 62 failing tests).

**Root cause**: `src/__tests__/vitest.setup.ts` installed a `MemoryStorage` polyfill only when `globalThis.localStorage` was `== null`. With the freshly-installed `happy-dom@20.6.0` + current Node, `globalThis.localStorage` is defined but as an incomplete stub missing methods like `.clear()`. The null-check silently skipped the polyfill, leaving the broken stub in place.

**Fix**: Changed the guard from `value == null` to a capability check — `typeof value?.clear === 'function'` — so the polyfill installs whenever the platform-provided storage is unusable, not just when it's absent. See `src/__tests__/vitest.setup.ts`.

**Why it matters**: this was a pre-existing, unrelated-to-the-rewrite environment bug that happened to surface because `pnpm install` picked up a newer `happy-dom` than whatever was last used. Fixed because tests could not otherwise run at all.
