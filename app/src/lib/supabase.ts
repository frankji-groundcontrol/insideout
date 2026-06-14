import { createClient, type SupabaseClient } from '@supabase/supabase-js'

// SSR-safe Supabase 客户端工厂：不再缓存模块级单例（会跨 SSR 请求泄漏 session），
// 也不再读取 import.meta.env。客户端由 plugins/services.ts 在每个请求内按需构建并注入到适配器。
export function createSupabaseClient(url: string, key: string): SupabaseClient {
  if (!url || !key) {
    throw new Error(
      'NUXT_PUBLIC_SUPABASE_URL and NUXT_PUBLIC_SUPABASE_ANON_KEY are required when using Supabase mode',
    )
  }
  return createClient(url, key)
}
