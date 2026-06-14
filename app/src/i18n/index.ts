import zhCN from './locales/zh-CN'
import enUS from './locales/en-US'

// SSR 安全：本模块不再调用 createI18n、不读取 localStorage。
// i18n 实例由 plugins/i18n.ts 在 Nuxt app 上下文中创建，
// localStorage 仅在客户端（import.meta.client）读取。
const STORAGE_KEY = 'juanleme-lang'

export const LANG_STORAGE_KEY = STORAGE_KEY
export const DEFAULT_LOCALE = 'zh-CN'

export { zhCN, enUS }
