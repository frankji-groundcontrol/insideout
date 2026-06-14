// SSR 安全的 localStorage 封装。
// 服务端（Nuxt SSR / Nitro）没有 localStorage：读操作返回 null、写操作静默跳过，
// 避免 "localStorage is not defined" 在服务端渲染时抛错。
// 浏览器与测试环境（happy-dom）下行为与原生 localStorage 完全一致。
function store(): Storage | null {
  return typeof localStorage !== 'undefined' ? localStorage : null
}

export const safeStorage = {
  getItem(key: string): string | null {
    return store()?.getItem(key) ?? null
  },
  setItem(key: string, value: string): void {
    store()?.setItem(key, value)
  },
  removeItem(key: string): void {
    store()?.removeItem(key)
  },
}
