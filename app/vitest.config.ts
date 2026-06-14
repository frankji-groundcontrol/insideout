import { fileURLToPath } from 'node:url'
import { defineVitestConfig } from '@nuxt/test-utils/config'

export default defineVitestConfig({
  test: {
    environment: 'happy-dom',
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    globals: true,
    // happy-dom@20 + 当前 Node 不再自动注入全局 localStorage/sessionStorage，
    // 而 store/组件/适配器测试依赖它们。该 setup 文件提供最小内存版兜底（仅在缺失时安装）。
    setupFiles: ['./src/__tests__/vitest.setup.ts'],
  },
  resolve: {
    // 保持 `@` → ./src 别名：5 个 supabase 适配器测试使用 vi.mock('@/lib/supabase')
    // 的字符串字面量，必须与被测源文件的 import 路径逐字匹配。
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
})
