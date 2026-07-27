import { fileURLToPath } from 'node:url'

// Internal (server-to-server) base URL for the Go API. The browser never
// talks to this directly — it hits same-origin /api/v1/**, which Nitro's
// routeRules proxy below forwards here, keeping the httpOnly auth cookies
// first-party (no CORS, no SameSite cross-site pitfalls). See
// docs/plans/2026-07-20-go-rewrite/04-frontend.md §1.
const apiInternalBase = process.env.NUXT_API_INTERNAL_BASE || 'http://127.0.0.1:8080/api/v1'

export default defineNuxtConfig({
  ssr: true,
  srcDir: 'src',
  compatibilityDate: '2025-06-01',

  modules: ['@pinia/nuxt', '@nuxtjs/tailwindcss'],

  // tokens.css defines the Ink & Seal semantic CSS vars and must load before
  // style.css's @tailwind directives so :root is defined before paint. It
  // was written but never added here — see docs/plans/2026-07-20-go-rewrite/04-frontend.md §4.
  css: ['~/assets/tokens.css', '~/style.css'],

  // Same-origin proxy to the Go API — see apiInternalBase above.
  routeRules: {
    '/api/v1/**': { proxy: `${apiInternalBase}/**` },
  },

  // Keep the existing '@' alias resolving to the Nuxt source root (./src).
  // '~' already points at srcDir; we add '@' explicitly because the codebase uses '@/...'.
  alias: {
    '@': fileURLToPath(new URL('./src', import.meta.url)),
  },

  // Auto-import components by bare name (e.g. <BaseButton/>, <RoadmapItem/>, <WorkshopCard/>)
  // instead of the path-prefixed names. Existing templates keep working unchanged.
  components: [{ path: '~/components', pathPrefix: false }],

  app: {
    head: {
      htmlAttrs: { lang: 'en' },
      title: 'InsideOut',
      // Function form so the bare default title ('InsideOut', when no page
      // sets its own) isn't ALSO run through the template — a plain '%s ·
      // InsideOut' string doubles up on every page that doesn't override it.
      // Cast: Nuxt's runtime accepts a function here, but the installed head
      // types declare titleTemplate as `string` only.
      titleTemplate: ((titleChunk?: string): string =>
        titleChunk && titleChunk !== 'InsideOut' ? `${titleChunk} · InsideOut` : 'InsideOut') as unknown as string,
      meta: [
        { charset: 'UTF-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1.0' },
      ],
      link: [
        { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' },
        // Fonts are referenced, never bundled. The Ink & Seal display serif
        // (Noto Serif SC — Song/Mincho for headings + chop labels) loads from
        // Google Fonts as progressive enhancement. The body sans (Alibaba
        // PuHuiTi) is named in the tailwind stack but NOT hosted here — it is
        // proprietary "free to use", not redistributable (see README); visitors
        // without it fall through to Noto Sans SC and the platform CJK face.
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
        {
          rel: 'stylesheet',
          href: 'https://fonts.googleapis.com/css2?family=Noto+Serif+SC:wght@500;600;700&display=swap',
        },
      ],
      // FOUC 修复：在首屏绘制前根据 localStorage 中的 juanleme-theme 设置 <html> 的 dark 类，
      // 避免 light→dark 闪烁以及 SSR/客户端 class 水合不一致。ThemeToggle 的 onMounted 仍会同步响应式状态。
      // No saved choice → follow the OS (prefers-color-scheme); an explicit saved choice always wins.
      script: [
        {
          innerHTML:
            "try{var t=localStorage.getItem('juanleme-theme');var d=t?t==='dark':matchMedia('(prefers-color-scheme:dark)').matches;document.documentElement.classList.toggle('dark',d)}catch(e){}",
          tagPosition: 'head',
        },
      ],
    },
  },

  runtimeConfig: {
    // Server-only: base URL the SSR process uses to call the Go API
    // directly (skipping the browser-facing proxy). See
    // src/services/api/http.ts.
    apiInternalBase,
    public: {},
  },

  // @nuxtjs/tailwindcss auto-detects content paths AND auto-wires PostCSS (tailwindcss + autoprefixer).
  // We keep an explicit tailwind.config.js so darkMode:'class' and widened globs are pinned.

  typescript: {
    // Surface type errors but do not block dev server. Set true if CI should fail on type errors.
    typeCheck: false,
  },
})
