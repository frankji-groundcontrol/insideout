# BUG-002: `BaseInput`'s default `id` used `Math.random()`, causing SSR hydration mismatches

**Found**: 2026-07-20, during the InsideOut rewrite (P5), verifying the new login/register pages in a browser via Playwright — visible as a Vue hydration-mismatch console error on first load.

**Symptom**: `[Vue warn]: Hydration attribute mismatch` on every `<label for>`/`<input id>` pair rendered by an unlabeled-`id` `BaseInput`, plus a top-level `Hydration completed but contains mismatches` error.

**Root cause**: `BaseInput.vue`'s prop default was `id: () => \`input-${Math.random().toString(36).substring(2, 9)}\``. `Math.random()` produces a different value on the server render and the client hydration render, so the server-rendered `id`/`for` never matches what the client expects. This existed before the rewrite but never manifested because no page used `BaseInput` without an explicit `id` — the old `login.vue` used raw `<input>` elements, not `BaseInput`. The new `login.vue`/`register.vue` adopted `BaseInput` properly, surfacing the latent bug.

**Fix**: Replaced `Math.random()` with Vue 3.5's `useId()`, which is SSR-stable by design (matching IDs across server render and client hydration). See `src/components/common/BaseInput.vue`.

**Why it matters**: any future component relying on `Math.random()`, `Date.now()`, or similar non-deterministic values for SSR-rendered attributes will hit the same class of bug. Prefer `useId()` for auto-generated DOM ids in this codebase going forward.
