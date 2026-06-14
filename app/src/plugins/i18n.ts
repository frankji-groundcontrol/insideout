import { createI18n } from 'vue-i18n'
import { zhCN, enUS, LANG_STORAGE_KEY, DEFAULT_LOCALE } from '@/i18n'

export default defineNuxtPlugin((nuxtApp) => {
  // SSR 阶段无 localStorage，默认 zh-CN；仅在客户端读取已保存的语言偏好。
  let locale: string = DEFAULT_LOCALE
  if (import.meta.client) {
    const saved = localStorage.getItem(LANG_STORAGE_KEY)
    if (saved === 'en-US' || saved === 'zh-CN') {
      locale = saved
    }
  }

  const i18n = createI18n({
    legacy: false,
    locale,
    fallbackLocale: 'zh-CN',
    globalInjection: true,
    messages: {
      'zh-CN': zhCN,
      'en-US': enUS,
    },
  })

  nuxtApp.vueApp.use(i18n)
})
