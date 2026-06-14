import { fileURLToPath } from 'node:url'

export default defineNuxtConfig({
  ssr: true,
  srcDir: 'src',
  compatibilityDate: '2025-06-01',

  modules: ['@pinia/nuxt', '@nuxtjs/tailwindcss'],

  // Global stylesheet (was imported in main.ts). @tailwind directives processed by the tailwind module.
  css: ['~/style.css'],

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
      title: 'app',
      meta: [
        { charset: 'UTF-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1.0' },
      ],
      link: [{ rel: 'icon', type: 'image/svg+xml', href: '/vite.svg' }],
      // FOUC 修复：在首屏绘制前根据 localStorage 中的 juanleme-theme 设置 <html> 的 dark 类，
      // 避免 light→dark 闪烁以及 SSR/客户端 class 水合不一致。ThemeToggle 的 onMounted 仍会同步响应式状态。
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
    // No private/server-only secrets in this app today. (Anthropic/Vault secrets live in the
    // Supabase Edge Function, not the Nuxt runtime.) Add server-only keys here later if needed.
    public: {
      apiMode: '', // NUXT_PUBLIC_API_MODE         (was VITE_API_MODE)
      bundleAuth: '', // NUXT_PUBLIC_BUNDLE_AUTH       (was VITE_BUNDLE_AUTH)
      bundleData: '', // NUXT_PUBLIC_BUNDLE_DATA       (was VITE_BUNDLE_DATA)
      bundleAiExport: '', // NUXT_PUBLIC_BUNDLE_AI_EXPORT  (was VITE_BUNDLE_AI_EXPORT)
      supabaseUrl: '', // NUXT_PUBLIC_SUPABASE_URL      (was VITE_SUPABASE_URL)
      supabaseAnonKey: '', // NUXT_PUBLIC_SUPABASE_ANON_KEY (was VITE_SUPABASE_ANON_KEY)
    },
  },

  // @nuxtjs/tailwindcss auto-detects content paths AND auto-wires PostCSS (tailwindcss + autoprefixer).
  // We keep an explicit tailwind.config.js so darkMode:'class' and widened globs are pinned.

  typescript: {
    // Surface type errors but do not block dev server. Set true if CI should fail on type errors.
    typeCheck: false,
  },
})
