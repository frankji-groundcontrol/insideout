# BUG-011: SSR always renders `zh-CN`, client hydrates to saved `en-US` — locale hydration mismatch on every page

**Found**: 2026-07-23, while visually verifying the Ink & Seal Assembly landing (`src/pages/index.vue`) in the dev preview — the console showed ~360 `Hydration text mismatch` warnings spanning the `NavBar` (layout) and every section of the page.

**Symptom**: `[Vue warn]: Hydration text mismatch` on every localized string. Server-rendered HTML is Chinese (`把想法，一件件搭出来。`, `开始旅程`); the client then re-renders in English (`Build your idea, piece by piece.`, `Start Journey`), so the two trees disagree. It resolves to English after hydration, but the mismatch floods the console and defeats the point of SSR for localized text.

**Root cause**: `src/plugins/i18n.ts` reads the saved locale from `localStorage`, which does not exist during SSR. So the server always falls back to `DEFAULT_LOCALE = 'zh-CN'` (`src/i18n/index.ts`), while the client reads `localStorage['juanleme-lang']` and — if the user previously chose English — hydrates to `'en-US'`. The plugin's own comment documents this: "SSR 阶段无 localStorage，默认 zh-CN；仅在客户端读取已保存的语言偏好。" This is structural and predates the landing redesign (it shows in the layout `NavBar`, which the landing work did not touch); the redesign only made it loud because the page is entirely localized text.

**Why it matters**: the whole app ships SSR HTML in the wrong language for any visitor whose saved preference is `en-US`, and every localized page logs hydration mismatches. The locale preference needs to be readable on the server to fix it properly.

**Candidate fix (not applied)**: store the locale in a **cookie** instead of (or mirrored from) `localStorage`, so SSR can read it via `useCookie()` and render the matching language on the server; keep `localStorage` in sync for legacy clients. Alternative: gate localized chrome behind `<ClientOnly>` — but that forfeits SSR for the nav/landing copy, so the cookie approach is preferred. Out of scope for the landing redesign; recorded for a dedicated i18n-SSR pass.
