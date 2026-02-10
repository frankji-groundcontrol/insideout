import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN'
import enUS from './locales/en-US'

const STORAGE_KEY = 'juanleme-lang'
const DEFAULT_LOCALE = 'zh-CN'

const savedLocale = localStorage.getItem(STORAGE_KEY)
const locale = savedLocale === 'en-US' || savedLocale === 'zh-CN' ? savedLocale : DEFAULT_LOCALE

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

export const LANG_STORAGE_KEY = STORAGE_KEY
export default i18n
