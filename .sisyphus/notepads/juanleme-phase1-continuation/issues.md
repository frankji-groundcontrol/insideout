# Issues & Gotchas

## 2026-02-10 Task: Initial Analysis
- WorkshopDetailView.vue has TWO script blocks (line 1-38 `<script setup>` and line 40-42 `<script>` with `computed` import). Must consolidate into single `<script setup>`.
- LoginPage redirects to `/` not `/dashboard` — needs fixing in Task 2
- NavBar has local `isLoggedIn` ref not synced to any store — needs fixing in Task 2
- Mock data has emojis in node titles (data.ts) — these are DATA not UI, may keep or clean
- tsconfig.app.json `erasableSyntaxOnly: true` means NO TypeScript enums — use string literal unions or `as const` objects
